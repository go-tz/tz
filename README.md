# Go packages for the IANA time zone database
This repository contains a Go module with packages for working with the [IANA time zone database](https://en.wikipedia.org/wiki/Tz_database) (tzdb), about which Wikipedia says:

> The tz database is a collaborative compilation of information about the world's time zones and rules for observing daylight saving time,
> primarily intended for use with computer programs and operating systems.
> Paul Eggert has been its editor and maintainer since 2005, with the organizational backing of ICANN.
> The tz database is also known as tzdata, the zoneinfo database or the IANA time zone database (after the Internet Assigned Numbers Authority),
> and occasionally as the Olson database, referring to the founding contributor, Arthur David Olson.

## Roadmap

- [x] Parse tzdb data files: Implemented in package [tzdb/tzfile](./tzdb/tzfile).
- [x] Download tzdb releases from IANA: Implemented in package [tzdb/ianadist](./tzdb/ianadist).
- [x] Read and write TZif files: Implemented in package [tzdb/tzif](./tzdb/tzif).
- [ ] Compile tzdb data files to TZif files, binary-compatible with the reference [timezone compiler](https://data.iana.org/time-zones/code/zic.8.txt).
- [ ] Distribution mechanism for TZif data, see [Time Zone Data Distribution Service](https://datatracker.ietf.org/wg/tzdist/documents/).

## Resources
- [https://github.com/eggert/tz](https://github.com/eggert/tz): Original time zone database and code
- [https://data.iana.org/time-zones](https://data.iana.org/time-zones): IANA file server for the time zone database
- [RFC6557](https://datatracker.ietf.org/doc/html/rfc6557): Procedures for Maintaining the Time Zone Database

The following pages are part of [eggert/tz](https://github.com/eggert/tz) and conveniently hosted by IANA:

- [theory.html: Theory and pragmatics of the tz code and data](https://data.iana.org/time-zones/code/theory.html)
- [tz-art.html: Time and the Arts](https://data.iana.org/time-zones/code/tz-art.html)
- [tz-how-to.html: How to Read the tz Database Source Files](https://data.iana.org/time-zones/code/tz-how-to.html)

You can also find a copy of those pages in the [tzdb/docs](./tzdb/docs) directory of this repository.

## License
Unless otherwise specified, all files in this repository are licensed under the [MIT license](LICENSE).

- The IANA time zone database is in the public domain.
  This project redistributed portions of the database and contains derived work 
  in the form of documentation which heavily quotes the documentation of the database.
- `tzdb/docs` contains unmodified copies of documentation from the tzdb release 2024b,
  [which is in the public domain](https://data.iana.org/time-zones/tzdb-2024b/LICENSE).
- Package [tzdb/tzif](./tzdb/tzif) implements the Time Zone Information Format (TZif) as specified in [RFC 8536](https://tools.ietf.org/html/rfc8536) authored by Arthur David Olson, Paul Eggert and Kenneth Murchison.
  It includes an unmodified copy of the RFC for documentation purposes and also contains samples from the RFC as part of the test suite. Additionally, comments and documentation in the code are derived from the RFC where appropriate.

We have made every effort to ensure that this project complies with applicable copyright laws and licensing terms. 
If you believe there are any issues or oversights regarding the licensing or use of any content in this repository, we would greatly appreciate it if you could open a GitHub issue to bring it to our attention.
