# shiplabel-lint

A command-line linter for shipping label batch files. It catches malformed
addresses, weights, dimensions, and service codes before you feed a batch
into a carrier API or a label printer, and it tells you exactly which line
and column is wrong.

## The problem this solves

Most shipping APIs will happily accept a batch file, reject the third label
in it, and tell you `400 Bad Request: invalid weight`. You then get to guess
which of your forty labels has the bad weight field, and the API rarely
tells you it was actually `1.5lbs` when it wanted `1.5 lb`. This tool checks
a batch file locally first and points at the exact spot, the same way a
compiler error does, instead of a vague field name.

## The format

A batch file is a sequence of labels. Each label is a block of `key: value`
lines; blank lines separate one label from the next. Lines starting with
`#` are comments.

```
to: 742 Evergreen Terrace, Springfield, IL 62704
from: 1 Amazon Way, Reno, NV 89501
weight: 3.2 lb
dims: 14x10x6 in
service: priority
```

Recognized fields:

| field     | required | format                                       |
|-----------|----------|-----------------------------------------------|
| `to`      | yes      | free text ending in a US zip code             |
| `from`    | yes      | free text ending in a US zip code             |
| `weight`  | yes      | a number and a unit: `lb`, `oz`, `kg`, or `g` |
| `dims`    | yes      | `LxWxH` and a unit: `in` or `cm`              |
| `service` | no       | one of `ground`, `priority`, `express`, `overnight` (defaults to `ground`) |

Fields outside this list are left alone: a batch exported from some other
system might carry a `carrier:` reference number or an internal `notes:`
field, and this tool has no opinion on those. Run with `--strict` to reject
anything outside the five fields above instead.

## Service limits

Once `weight` and `dims` are individually well-formed, they're also checked
against the limit for whatever `service` the label names (`ground` if the
field is left out):

| service     | max weight | max longest side |
|-------------|-----------:|------------------:|
| `ground`    |     150 lb |            108 in |
| `priority`  |      70 lb |            108 in |
| `express`   |     150 lb |            108 in |
| `overnight` |      70 lb |             96 in |

Weight and dimensions are converted to pounds and inches before the check,
so a `weight: 80 kg` label is compared against the limit the same as an
equivalent `weight: 176.4 lb` would be. If `service` itself doesn't name a
known service, that's reported on its own and the weight/dims limit check
is skipped for that label, since there's no limit to check against.

## Usage

```
shiplabel-lint testdata/sample.labels
shiplabel-lint --strict testdata/sample.labels
```

Pass `--json` to get the same information as structured data on stdout instead
of the compiler-style text on stderr, for feeding into another tool instead of
a terminal:

```
shiplabel-lint --json testdata/sample.labels
```

```json
{
  "files": [
    {
      "file": "testdata/sample.labels",
      "ok": false,
      "labels": 2,
      "errors": [
        {
          "line": 7,
          "col": 5,
          "message": "address \"221B Baker St, London\" does not end with a valid US zip code (expected 5 digits or 5+4, like \"62704\" or \"62704-1234\")",
          "source_line": "to: 221B Baker St, London"
        },
        {
          "line": 9,
          "col": 9,
          "message": "invalid weight \"1.5lbs\": expected a number followed by a unit (lb, oz, kg, or g), like \"4.5 lb\"",
          "source_line": "weight: 1.5lbs"
        }
      ]
    }
  ],
  "error_count": 2
}
```

The exit status is the same either way: 1 if any file has errors, 2 on usage
or file errors.

Pass `--fix` to rewrite the files in place before linting. It only touches
`weight` and `dims` values that are currently invalid and that it can make
valid without guessing: missing space before the unit (`3.2lb`), plural or
upper-case units (`1.5lbs`, `4 KG`), and upper-case or spaced `x` in
dimensions (`12 X 8 X 6in`). Each rewrite is printed to stderr with its
position, and anything left over is reported as usual, so a value with no unit
at all is still an error. Line endings and the rest of the file are kept.

```
shiplabel-lint --fix batch.labels
```

You can also pass more than one file, or a directory, in a single run. A
directory argument is walked for every `*.labels` file underneath it
(directories starting with `.` are skipped), so a whole batch drop can be
checked at once:

```
shiplabel-lint batch1.labels batch2.labels
shiplabel-lint ./incoming/
```

Each file is reported independently with its own filename in every error,
and the exit status reflects errors across all of them combined.

`testdata/sample.labels` has one good label and one with two mistakes.
Running the tool against it prints:

```
sample.labels:7:5: address "221B Baker St, London" does not end with a valid US zip code (expected 5 digits or 5+4, like "62704" or "62704-1234")
  7 | to: 221B Baker St, London
    |     ^

sample.labels:9:9: invalid weight "1.5lbs": expected a number followed by a unit (lb, oz, kg, or g), like "4.5 lb"
  9 | weight: 1.5lbs
    |         ^

2 error(s) found
```

The tool exits with status 1 when it finds errors, and 2 on usage or file
errors, so it plugs into a pre-send check in a build step. On a clean file
it prints a one-line summary and exits 0:

```
good.labels: 3 label(s) OK
```

## Building

```
go build -o shiplabel-lint .
```

No third-party dependencies; the standard library is enough for parsing,
regex validation, and file I/O.
