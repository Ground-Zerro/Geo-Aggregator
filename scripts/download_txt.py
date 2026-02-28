import os
import re
import time
import urllib.error
import urllib.request
import yaml

CN_PATTERN = re.compile(r'^cn$|^cn[-_]|[-_]cn$|[-_]cn[-_]|china|geolocation-cn', re.IGNORECASE)
PREFIX_PATTERN = re.compile(r'^(full:|domain:|regexp:|keyword:|ext:\S+:)')
AT_ANNOTATION = re.compile(r':@(\S+)$')
VALID_DOMAIN = re.compile(r'^[a-zA-Z0-9.\-]+$')
LABEL_RE = re.compile(r'\\\.([a-zA-Z0-9][a-zA-Z0-9-]*)')
STRICT_LABEL = re.compile(r'^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$')

RETRIES = 3
TIMEOUT = 30


def fetch(url: str, dest: str) -> bool:
    os.makedirs(os.path.dirname(dest), exist_ok=True)
    for attempt in range(1, RETRIES + 1):
        try:
            urllib.request.urlretrieve(url, dest)
            return True
        except (urllib.error.URLError, OSError) as e:
            print(f"  [попытка {attempt}/{RETRIES}] ошибка: {e}")
            if attempt < RETRIES:
                time.sleep(TIMEOUT)
    print(f"  ПРОПУЩЕНО: {url}")
    return False


def is_cn_annotation(annotation: str) -> bool:
    parts = re.split(r',@', annotation)
    return any(CN_PATTERN.search(p) for p in parts if p and not p.startswith('!'))


def extract_domain_from_regex(pattern: str) -> str:
    parts = LABEL_RE.findall(pattern)
    domain_parts = []
    for part in reversed(parts):
        if not STRICT_LABEL.match(part):
            break
        domain_parts.insert(0, part)
    return '.'.join(domain_parts) if len(domain_parts) >= 2 else ''


def process_dlc_rule(rule: str) -> str:
    cleaned = PREFIX_PATTERN.sub('', str(rule).strip())
    if not cleaned or cleaned.startswith('#'):
        return ''
    m = AT_ANNOTATION.search(cleaned)
    if m:
        if is_cn_annotation(m.group(1)):
            return ''
        cleaned = cleaned[:m.start()]
    if not cleaned:
        return ''
    if not VALID_DOMAIN.match(cleaned):
        return extract_domain_from_regex(cleaned)
    return cleaned


def write_entries(entries: list, dest: str) -> None:
    os.makedirs(os.path.dirname(dest), exist_ok=True)
    with open(dest, 'w', encoding='utf-8') as f:
        f.write('\n'.join(entries) + '\n')


# ── v2fly/domain-list-community ──────────────────────────────────────────────

print("=== v2fly/domain-list-community ===")
yaml_path = "temp/dlc_plain.yml"
if fetch(
    "https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat_plain.yml",
    yaml_path,
):
    with open(yaml_path, 'r', encoding='utf-8') as f:
        dlc = yaml.safe_load(f)

    skipped_cn = 0
    for entry in dlc['lists']:
        tag = str(entry['name']).lower()
        if CN_PATTERN.search(tag):
            skipped_cn += 1
            continue
        entries = set()
        for rule in entry.get('rules', []):
            domain = process_dlc_rule(rule)
            if domain:
                entries.add(domain)
        if entries:
            write_entries(sorted(entries), f"temp/site/v2fly/{tag}")

    os.remove(yaml_path)
    print(f"  Обработано тегов: {len(dlc['lists'])}, пропущено CN: {skipped_cn}")


# ── Loyalsoldier/v2ray-rules-dat ─────────────────────────────────────────────

LOYALSOLDIER_FILES = [
    ("proxy-list.txt",  "proxy"),
    ("gfw.txt",         "gfw"),
    ("reject-list.txt", "reject"),
    ("direct-list.txt", "direct"),
    ("greatfire.txt",   "greatfire"),
    ("win-spy.txt",     "win-spy"),
    ("win-update.txt",  "win-update"),
    ("win-extra.txt",   "win-extra"),
]

print("=== Loyalsoldier/v2ray-rules-dat ===")
BASE = "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download"
for fname, tag in LOYALSOLDIER_FILES:
    fetch(f"{BASE}/{fname}", f"temp/site/loyalsoldier/{tag}")


# ── itdoginfo/allow-domains ──────────────────────────────────────────────────

print("=== itdoginfo/allow-domains ===")
BASE = "https://raw.githubusercontent.com/itdoginfo/allow-domains/main/Russia"
fetch(f"{BASE}/inside-raw.lst", "temp/site/itdoginfo/russia-inside")
fetch(f"{BASE}/outside-raw.lst", "temp/site/itdoginfo/russia-outside")


# ── antifilter.download ──────────────────────────────────────────────────────

print("=== antifilter.download ===")
fetch("https://antifilter.download/list/allyouneed.lst",          "temp/ip/antifilter/antifilter")
fetch("https://community.antifilter.download/list/community.lst", "temp/ip/antifilter/antifilter-community")
fetch("https://community.antifilter.download/list/domains.lst",   "temp/site/antifilter/antifilter-community")

print("=== Готово ===")
