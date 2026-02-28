import os
import re

CN_PATTERN = re.compile(r'^cn$|^cn[-_]|[-_]cn$|[-_]cn[-_]|china|geolocation-cn', re.IGNORECASE)
PREFIX_PATTERN = re.compile(r'^(full:|domain:|regexp:|keyword:|ext:\S+:)')
CN_AT = re.compile(r':@(?:[^@]*,@)*(?:cn|geolocation-cn)(?:,@[^@]*)*$', re.IGNORECASE)
AT_ANNOTATION = re.compile(r':@\S+$')
REGEX_CHARS = re.compile(r'[\\^$+*?{}\[\]()|]')


def is_cn_tag(tag: str) -> bool:
    return bool(CN_PATTERN.search(tag))


def process_line(line: str) -> str:
    line = line.strip()
    if not line or line.startswith('#'):
        return ''
    line = PREFIX_PATTERN.sub('', line).strip()
    if CN_AT.search(line):
        return ''
    line = AT_ANNOTATION.sub('', line).strip()
    if not line or REGEX_CHARS.search(line):
        return ''
    return line


V2DAT_PREFIX = re.compile(r'^[^_]+_')


def normalize_tag(fname: str) -> str:
    tag = fname.lower()
    if tag.endswith('.txt'):
        tag = tag[:-4]
    tag = V2DAT_PREFIX.sub('', tag)
    return tag


def collect_tags(base_dir: str) -> dict:
    tags = {}
    for root, dirs, files in os.walk(base_dir):
        dirs.sort()
        for fname in sorted(files):
            if fname.startswith('.'):
                continue
            tag = normalize_tag(fname)
            if is_cn_tag(tag):
                continue
            filepath = os.path.join(root, fname)
            entries = set()
            with open(filepath, 'r', encoding='utf-8', errors='ignore') as f:
                for line in f:
                    entry = process_line(line)
                    if entry:
                        entries.add(entry)
            if entries:
                tags.setdefault(tag, set()).update(entries)
    return tags


os.makedirs('source', exist_ok=True)

all_tags = {}
for subdir in ['temp/ip', 'temp/site']:
    if os.path.isdir(subdir):
        for tag, entries in collect_tags(subdir).items():
            all_tags.setdefault(tag, set()).update(entries)

for tag, entries in sorted(all_tags.items()):
    sorted_entries = sorted(entries)
    out_path = os.path.join('source', f'{tag}.txt')
    with open(out_path, 'w', encoding='utf-8') as f:
        f.write('\n'.join(sorted_entries) + '\n')
    print(f'Written: {out_path} ({len(sorted_entries)} entries)')

print(f'Total tags: {len(all_tags)}')
