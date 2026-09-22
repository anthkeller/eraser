#!/usr/bin/env python3
"""Normalize public state data-broker registries into Eraser YAML."""

import argparse
import csv
import re
from pathlib import Path
from urllib.parse import urlparse

import openpyxl
import yaml


def text(value):
    return "" if value is None else str(value).strip()


def slug(value):
    value = re.sub(r"[^a-z0-9]+", "-", value.lower()).strip("-")
    return value[:80] or "unnamed-broker"


def web_url(value):
    value = text(value)
    if not value:
        return ""
    match = re.search(r"https?://[^\s,;)]+", value)
    if match:
        return match.group(0).rstrip(".")
    if re.match(r"^(www\.)?[a-z0-9.-]+\.[a-z]{2,}(/.*)?$", value, re.I):
        return "https://" + value
    return ""


def domain(value):
    url = web_url(value)
    if not url:
        return ""
    return urlparse(url).netloc.lower().removeprefix("www.")


def email(value):
    value = text(value).lower()
    match = re.search(r"[a-z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-z0-9.-]+\.[a-z]{2,}", value)
    return match.group(0) if match else ""


def read_csv(path):
    with open(path, encoding="utf-8-sig", newline="") as stream:
        return list(csv.DictReader(stream))


def read_oregon_xlsx(path):
    book = openpyxl.load_workbook(path, read_only=True, data_only=True)
    sheet = book.active
    rows = sheet.iter_rows(values_only=True)
    for _ in range(3):
        next(rows)
    headers = [text(v) for v in next(rows)]
    return [dict(zip(headers, row)) for row in rows if any(v is not None for v in row)]


class Catalog:
    def __init__(self, existing_path):
        existing = yaml.safe_load(Path(existing_path).read_text()) or {}
        self.existing_ids = set()
        self.existing_by_name = {}
        for broker in existing.get("brokers", []):
            broker_id = text(broker.get("id"))
            self.existing_ids.add(broker_id)
            self.existing_by_name[slug(text(broker.get("name")))] = broker_id
        self.records = {}

    def add(self, *, name, website="", contact_email="", opt_out_url="", source,
            record_id="", updated_at="", risk_flags=(), tags=()):
        name = text(name)
        if not name:
            return
        website = web_url(website)
        opt_out_url = web_url(opt_out_url)
        # Legal-name matching is deliberately primary. Privacy portals often
        # use shared OneTrust/DataGrail domains, and some older registry rows
        # contain a related company's website; domain-only matching would join
        # unrelated legal entities.
        key = slug(name)
        broker_id = self.existing_by_name.get(key)
        if not broker_id:
            broker_id = slug(name)
        record = self.records.setdefault(key, {
            "id": broker_id, "name": name, "email": "", "website": "",
            "opt_out_url": "", "region": "us", "category": "marketing",
            "tags": [], "risk_flags": [], "registry_ids": {}, "sources": [],
        })
        if not record["email"]:
            record["email"] = email(contact_email)
        if not record["website"]:
            record["website"] = website
        if not record["opt_out_url"]:
            record["opt_out_url"] = opt_out_url
        record["tags"] = sorted(set(record["tags"]) | set(filter(None, tags)))
        record["risk_flags"] = sorted(set(record["risk_flags"]) | set(filter(None, risk_flags)))
        if record_id:
            record["registry_ids"].setdefault(source, text(record_id))
        source_entry = {"registry": source}
        if record_id:
            source_entry["record_id"] = text(record_id)
        if updated_at:
            source_entry["updated_at"] = text(updated_at)
        if source_entry not in record["sources"]:
            record["sources"].append(source_entry)

    def output(self):
        used = set(self.existing_ids)
        result = []
        for record in sorted(self.records.values(), key=lambda item: item["name"].lower()):
            if record["id"] in self.existing_ids:
                pass
            elif record["id"] in used:
                suffix = slug(domain(record["website"]) or next(iter(record["registry_ids"].values()), "record"))
                record["id"] = f"{record['id']}-{suffix}"[:100]
            used.add(record["id"])
            for field in ("email", "website", "opt_out_url", "tags", "risk_flags", "registry_ids"):
                if not record[field]:
                    record.pop(field)
            result.append(record)
        return {"brokers": result}


def yes(value):
    return text(value).lower().startswith("yes")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--california", required=True)
    parser.add_argument("--oregon", required=True)
    parser.add_argument("--oregon-collected", required=True)
    parser.add_argument("--oregon-optout", required=True)
    parser.add_argument("--vermont", required=True)
    parser.add_argument("--existing", default="data/brokers.yaml")
    parser.add_argument("--output", default="data/registry-brokers.yaml")
    args = parser.parse_args()
    catalog = Catalog(args.existing)

    for row in read_csv(args.california):
        flags = []
        flag_columns = {
            "minors": "Data broker collects personal information of minors:",
            "account-credentials": "Data broker collects consumers’ account logins or numbers with security codes that grant access to third‑party accounts",
            "government-id": "Data broker collects consumers’ government‑issued identification numbers used to verify an individual’s identity",
            "biometric": "Data broker collects consumers' biometric data",
            "precise-geolocation": "Data broker collects consumers' precise geolocation",
            "reproductive-health": "Data broker collects consumers’ reproductive health care data:",
        }
        for flag, column in flag_columns.items():
            if yes(row.get(column)):
                flags.append(flag)
        catalog.add(name=row.get("Data broker name:"), website=row.get("Data broker primary website:"),
                    contact_email=row.get("Data broker primary contact email address:"),
                    opt_out_url=row.get("Data broker's primary website that contains details on how consumers can exercise their CA Consumer Privacy rights:"),
                    source="california-2026", risk_flags=flags, tags=["state-registered"])

    for row in read_oregon_xlsx(args.oregon):
        flags = ["minors"] if yes(row.get("Data of a Known Child")) else []
        catalog.add(name=row.get("Full Legal Name"), website=row.get("Website"),
                    contact_email=row.get("Data Broker Email Address"), opt_out_url=row.get("Consumer Rights Link"),
                    source="oregon-2026", record_id=row.get("Registration Number"), updated_at=row.get("Submitted On"),
                    risk_flags=flags, tags=["state-registered"])

    optout = {text(r.get("LICENSE NO.")): r for r in read_csv(args.oregon_optout)}
    for row in read_csv(args.oregon_collected):
        license_id = text(row.get("LICENSE NO."))
        contact = optout.get(license_id, {})
        flags = []
        for flag, column in (("date-of-birth", "DATA COLLECT - DATE OF BIRTH"),
                             ("biometric", "DATA COLLECT - BIOMETRIC INFORMATION"),
                             ("government-id", "DATA COLLECT - SSN/GOVERNMENT ID")):
            if "collect" in text(row.get(column)).lower() and "does not" not in text(row.get(column)).lower():
                flags.append(flag)
        opt_url = web_url(contact.get("OPT OUT OTHER METHODS")) or web_url(contact.get("OPT OUT NARRATIVE"))
        catalog.add(name=row.get("FACILITY NAME"), website=contact.get("OPT OUT WEBSITE"),
                    contact_email=contact.get("OPT OUT EMAIL"), opt_out_url=opt_url,
                    source="oregon-dfr-2025", record_id=license_id, updated_at=row.get("UPDATE DATE"),
                    risk_flags=flags, tags=["state-registered", "proxy-opt-out"] if yes(row.get("OPT OUT OF BY PROXY")) else ["state-registered"])

    for row in read_csv(args.vermont):
        method = text(row.get("a. What was the method for requesting an opt-out?"))
        catalog.add(name=row.get("Data Broker Name:"), website=row.get("Primary Internet Address:"),
                    contact_email=row.get("Email Address:"), opt_out_url=web_url(method),
                    source="vermont", record_id=row.get("Registration ID:"), updated_at=row.get("Created Date:"),
                    risk_flags=["minors"] if yes(row.get("7. Does the data broker have actual knowledge that it possesses the brokered personal information of minors:")) else [],
                    tags=["state-registered"])

    output = catalog.output()
    Path(args.output).write_text(yaml.safe_dump(output, sort_keys=False, allow_unicode=True), encoding="utf-8")
    print(f"wrote {len(output['brokers'])} normalized brokers to {args.output}")


if __name__ == "__main__":
    main()
