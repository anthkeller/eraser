# Broker registry sources

`registry-brokers.yaml` was generated on 2026-09-22 from these supplied public
registry exports:

| Source key | Input export | Source rows represented |
| --- | --- | ---: |
| `california-2026` | California Data Broker Registry 2026.csv | 579 |
| `oregon-2026` | Data Broker Registrationses - 2026-09-22_0147.xlsx | 445 |
| `oregon-dfr-2025` | dfcs_db_collected.csv + dfcs_db_optout.csv | 389 |
| `vermont` | vt-data-brokers.csv | 278 |

The counts are source memberships after same-name records were combined. One
normalized broker can cite more than one registry.

The importer preserves registration IDs, source dates, contact emails, primary
websites, consumer-rights links, proxy opt-out support, and selected sensitive
data flags. It does not import postal addresses, phone numbers, staff names, or
free-form registry narratives because Eraser does not need those fields to
route removal workflows.

The supplied Massachusetts `9-21-26.pdf` is a data-breach notification report,
not a data-broker registry. It was reviewed but excluded from the broker catalog.
The supplied `cocoon-database-log.txt` is unrelated application diagnostic data
and was also excluded.
