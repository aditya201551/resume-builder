"""Regenerate the attributed offline GeoNames city index. Run from any directory."""
import io
import json
from pathlib import Path
from urllib.request import urlopen
import zipfile

BASE = "https://download.geonames.org/export/dump/"


def download(name):
    with urlopen(BASE + name, timeout=60) as response:
        return response.read()


def main():
    countries = {}
    for line in download("countryInfo.txt").decode().splitlines():
        if line.startswith("#") or not line.strip():
            continue
        fields = line.split("\t")
        countries[fields[0]] = {
            "code": fields[0], "name": fields[4], "continent": fields[8], "aliases": [fields[1]]
        }
    for code, aliases in {
        "US": ["USA", "U.S.", "U.S.A.", "United States of America"],
        "GB": ["UK", "U.K.", "Great Britain", "England", "Scotland", "Wales", "Northern Ireland"],
        "KR": ["South Korea", "Republic of Korea"], "AE": ["UAE"], "CZ": ["Czechia"],
        "TW": ["Taiwan"], "VN": ["Viet Nam"], "RU": ["Russian Federation"],
        "TR": ["Türkiye"], "NL": ["The Netherlands"],
    }.items():
        countries[code]["aliases"] += aliases

    archive = zipfile.ZipFile(io.BytesIO(download("cities15000.zip")))
    cities = []
    for line in archive.read("cities15000.txt").decode().splitlines():
        fields = line.split("\t")
        if fields[8] not in countries:
            continue
        aliases = list(dict.fromkeys([fields[2]] + [a for a in fields[3].split(",") if a.isascii() and len(a) > 3]))[:10]
        cities.append({
            "id": fields[0], "name": fields[1], "aliases": aliases,
            "latitude": float(fields[4]), "longitude": float(fields[5]),
            "countryCode": fields[8], "population": int(fields[14] or 0),
        })
    cities.sort(key=lambda p: p["population"], reverse=True)
    selected = {city["id"]: city for city in cities[:1800]}
    for code in countries:
        selected.update({p["id"]: p for p in [c for c in cities if c["countryCode"] == code][:3]})

    offsets = {}
    for line in download("timeZones.txt").decode().splitlines()[1:]:
        fields = line.split("\t")
        if len(fields) >= 5:
            offsets.setdefault(fields[0], set()).add(float(fields[4]))
    for code, country in countries.items():
        country["utcOffsets"] = sorted(offsets.get(code, set()))

    data = {
        "source": "GeoNames cities15000 and countryInfo, CC BY 4.0, https://www.geonames.org/",
        "countries": sorted(countries.values(), key=lambda c: c["name"]),
        "places": list(selected.values()),
    }
    target = Path(__file__).resolve().parents[1] / "backend/internal/jobmap/places.json"
    target.write_text(json.dumps(data, ensure_ascii=False, separators=(",", ":")) + "\n", encoding="utf-8")
    print(f"Wrote {len(selected)} cities and {len(countries)} countries to {target}")


if __name__ == "__main__":
    main()
