import gzip
import json
import os

repo = os.environ['GEO_REPO']
branch = os.environ['GEO_BRANCH']

index = {}
for fname in sorted(os.listdir('source')):
    if not fname.endswith('.txt'):
        continue
    tag = fname[:-4]
    index[tag] = f"https://raw.githubusercontent.com/{repo}/{branch}/source/{fname}"

with open('database.json', 'w', encoding='utf-8') as f:
    json.dump(index, f, indent=2, ensure_ascii=False)

with open('database.json', 'rb') as f_in:
    with gzip.open('database.json.gz', 'wb', compresslevel=9) as f_out:
        f_out.write(f_in.read())

print(f'Index generated: {len(index)} entries')
