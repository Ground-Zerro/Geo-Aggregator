import json
import os
from datetime import datetime, timezone

repo = os.environ['GEO_REPO']
branch = os.environ['GEO_BRANCH']

with open('database.json', 'r', encoding='utf-8') as f:
    index = json.load(f)

updated = datetime.now(timezone.utc).strftime('%Y-%m-%d %H:%M UTC')

lines = [
    "# Geo-Aggregator",
    "",
    "Автономный агрегатор GeoIP и GeoSite данных. Объединяет мировые и российские базы в единые текстовые списки по категориям, обновляется ежедневно.",
    "",
    f"**Последнее обновление:** {updated}",
    "",
    "## Использование",
    "",
    "Прямая ссылка на категорию:",
    "```",
    f"https://raw.githubusercontent.com/{repo}/{branch}/source/<tag>.txt",
    "```",
    "",
    "Индекс всех категорий:",
    "```",
    f"https://raw.githubusercontent.com/{repo}/{branch}/database.json",
    f"https://raw.githubusercontent.com/{repo}/{branch}/database.json.gz",
    "```",
    "",
    "## Источники",
    "",
    "| Репозиторий | Данные |",
    "|---|---|",
    "| [Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat) | IP + домены (proxy, gfw, reject и др.) |",
    "| [v2fly/geoip](https://github.com/v2fly/geoip) | IP-диапазоны по странам и сервисам |",
    "| [v2fly/domain-list-community](https://github.com/v2fly/domain-list-community) | Домены (1400+ тегов) |",
    "| [runetfreedom/russia-v2ray-rules-dat](https://github.com/runetfreedom/russia-v2ray-rules-dat) | IP + домены РФ (заблокированные) |",
    "| [itdoginfo/allow-domains](https://github.com/itdoginfo/allow-domains) | Домены РФ (inside/outside) |",
    "| [antifilter.download](https://antifilter.download) | IP-адреса + домены (АнтиФильтр) |",
    "",
    "---",
    "",
    f"*Автоматически сгенерировано GitHub Actions · {len(index)} категорий*",
]

with open('README.md', 'w', encoding='utf-8') as f:
    f.write('\n'.join(lines) + '\n')

print(f'README updated: {len(index)} categories')
