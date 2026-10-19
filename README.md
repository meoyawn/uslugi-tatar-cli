# uslugi-tatar-cli

A Go CLI for submitting gas, electricity and water meter readings to
[Госуслуги Республики Татарстан](https://uslugi.tatarstan.ru/).

- `gas`: submit a configured gas meter reading.
- `main`: submit electricity, hot water and cold water readings for a saved ЖКХ account.
- `--dry-run`: validate the input and compare with the portal without submitting readings.

Requires Go 1.26+ and an authenticated Chrome session for the portal.

## Install

```console
go install github.com/meoyawn/uslugi-tatar-cli@latest
```

Or build from source:

```console
git clone https://github.com/meoyawn/uslugi-tatar-cli.git
cd uslugi-tatar-cli
go build -trimpath .
```

## Configure

Copy `meters.example.yaml` to a private `meters.yaml` outside the checkout, or
use the ignored `meters.yaml` filename locally. All example accounts, addresses,
UUIDs, serials, dates and readings are invented. Replace them with your own
saved-account identifiers and a reading dated in the current Moscow month.

Routing comes from the YAML contents, independently of the file name:

```yaml
uslugi:
  main:
    account: "0000000001"
    flat: "1"
    address: "ул ПРИМЕРНАЯ, дом 1"
    informer: "00000000-0000-4000-8000-000000000001"
  gas:
    account: "000000000001"
    meter: "100001"

2000-10-18:
  electricity: 1100
  hot: 110
  cold: 200
  gas: 123
```

For `gas`, use the 12-digit account and meter serial from your saved gas account.
The CLI selects that account, obtains its initials from the portal, and checks
the displayed account and meter before submitting. It preserves your default
saved-account preference.

For `main`, use your 10-digit ЖКХ account, apartment number, expected street/house
text and the informer UUID. The UUID is the `informer_id` parameter in the saved
ЖКХ card's `/personal-account/init-by-user-tsj` URL. The CLI checks the account,
address and apartment before opening its meters. Supported services are:

| YAML field | Portal service |
| --- | --- |
| electricity | Электроснабжение день |
| hot | Подогрев воды |
| cold | Холодное водоснабжение |

Gas and main routing are independent; each command requires only its own routing
and meter fields. Other supported meter fields may remain in the same file.

## Run

Flags go before the YAML path:

```console
uslugi-tatar-cli gas --dry-run /path/to/private/meters.yaml
uslugi-tatar-cli main --dry-run /path/to/private/meters.yaml
uslugi-tatar-cli gas /path/to/private/meters.yaml
uslugi-tatar-cli main /path/to/private/meters.yaml
```

`--profile Default` selects a Chrome profile by name or directory when automatic
discovery does not find the desired session.

Authentication uses [SweetCookie](https://github.com/steipete/sweetcookie/) to
read the existing Chrome cookies for the portal on every run. The CLI keeps
cookies in memory and does not print or save their values. No password or cookie
export is needed. On macOS, reading Chrome Safe Storage may prompt via Keychain.
Refresh your login in Chrome if the session expires.

## Validation and submission

The latest dated entry must be in the current `Europe/Moscow` calendar month
and cannot be in the future. Dates support `YYYY-MM-DD` and legacy `DD-MM-YYYY`.
Readings must be nonnegative whole YAML integers. Cumulative readings cannot
decrease against either local history or live portal values. Gas values must
fit six digits; main meter capacities are obtained from the portal.

Both commands report consumption since the preceding recorded reading. Missing
months produce a reminder and the consumption across the entire recorded gap,
including its date range and duration. Input files are never modified.

A dry run selects the account and opens its live meter form, then prints the
proposed changes. It never posts meter readings, though selecting a service can
require a workflow POST.

A normal `gas` run posts all six digits and the hidden total, requires the portal's
success acknowledgement, then separately reads back the last transmitted value.
Equal gas readings can be resubmitted. The YAML reading date is sent with gas;
the main form uses billing dates supplied by the portal.

A normal `main` run skips submission when all three values already match. When
values change, it preserves the original hidden fields, serialized `item[]`
metadata and fresh CSRF/session fields, then separately reads back the values
and meter serials. Neither command makes payments. Uncertain submissions are
never automatically retried; inspect the live values with a dry run before
retrying.

## Development

```console
go test -race ./...
go vet ./...
```

Tests run against local HTTP servers and entirely synthetic fixtures. Keep
personal meter files, browser exports, HAR captures and request logs out of
commits. Use invented accounts, addresses, UUIDs, cookies and readings in any
new tests or documentation.

Licensed under MIT. This project is not affiliated with the portal operator.
