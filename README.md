# Geo-Aggregator

Автономный агрегатор GeoIP и GeoSite данных. Объединяет мировые и российские базы в единые текстовые списки по категориям, обновляется ежедневно.

## Использование

Прямая ссылка на категорию (N — номер папки от 1, точный путь см. в database.json):
```
https://raw.githubusercontent.com/Ground-Zerro/Geo-Aggregator/main/source<N>/<tag>.txt
```

Индекс всех категорий:
```
https://raw.githubusercontent.com/Ground-Zerro/Geo-Aggregator/main/db/database.json
https://raw.githubusercontent.com/Ground-Zerro/Geo-Aggregator/main/db/database.json.gz
```

Geo-файлы для v2ray/v2fly:
```
https://raw.githubusercontent.com/Ground-Zerro/Geo-Aggregator/main/geodat/geosite_GA.dat
https://raw.githubusercontent.com/Ground-Zerro/Geo-Aggregator/main/geodat/geoip_GA.dat
```

## Источники

| Репозиторий | Данные |
|---|---|
| [Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat) | IP + домены (proxy, gfw, reject и др.) |
| [v2fly/geoip](https://github.com/v2fly/geoip) | IP-диапазоны по странам и сервисам |
| [v2fly/domain-list-community](https://github.com/v2fly/domain-list-community) | Домены (1400+ тегов) |
| [runetfreedom/russia-v2ray-rules-dat](https://github.com/runetfreedom/russia-v2ray-rules-dat) | IP + домены РФ (заблокированные) |
| [itdoginfo/allow-domains](https://github.com/itdoginfo/allow-domains) | Домены РФ (inside/outside) |
| [antifilter.download](https://antifilter.download) | IP-адреса + домены (АнтиФильтр) |

---

*Автоматически сгенерировано GitHub Actions · 1637 категорий · 2026-04-19 05:28 UTC*
