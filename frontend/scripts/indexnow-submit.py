#!/usr/bin/env python3
"""Отправляет все URL из sitemap.xml сайта в IndexNow (api.indexnow.org).

В отличие от site-001/finance-001, здесь НЕТ автодеплоя и deploy.sh — сервер
(/var/www/mv-004) не git-репозиторий, деплой только вручную: scp изменённых
файлов + npm run build + pm2 restart mv-004. Этот скрипт запускается
ОТДЕЛЬНО, вручную, ПОСЛЕ такого деплоя (когда сайт уже точно поднялся и
отдаёт свежий sitemap.xml). Использует только стандартную библиотеку
Python — на сервере нет pip-пакетов вроде requests.
"""
import json
import sys
import urllib.error
import urllib.request
import xml.etree.ElementTree as ET

DOMAIN = "men.an51.su"
KEY = "7e1d4b96869cb1df48c03010f8f35ef0"


def main() -> int:
    sitemap_url = f"https://{DOMAIN}/sitemap.xml"
    try:
        with urllib.request.urlopen(sitemap_url, timeout=15) as resp:
            xml_data = resp.read()
    except Exception as e:
        print(f"IndexNow: не удалось получить {sitemap_url}: {e}", file=sys.stderr)
        return 1

    ns = {"sm": "http://www.sitemaps.org/schemas/sitemap/0.9"}
    root = ET.fromstring(xml_data)
    urls = [el.text.strip() for el in root.findall(".//sm:loc", ns) if el.text]

    if not urls:
        print("IndexNow: sitemap.xml не содержит URL, отправка отменена.", file=sys.stderr)
        return 1

    payload = {
        "host": DOMAIN,
        "key": KEY,
        "keyLocation": f"https://{DOMAIN}/{KEY}.txt",
        "urlList": urls,
    }
    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        "https://api.indexnow.org/indexnow",
        data=data,
        headers={"Content-Type": "application/json; charset=utf-8"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            print(f"IndexNow: HTTP {resp.status}, отправлено {len(urls)} URL")
            return 0
    except urllib.error.HTTPError as e:
        body = e.read().decode(errors="replace")
        print(f"IndexNow: HTTP {e.code} — {body}", file=sys.stderr)
        return 1
    except Exception as e:
        print(f"IndexNow: ошибка запроса: {e}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
