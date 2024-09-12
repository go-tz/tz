# go-tz

## Packages

* [tzif](tzif): Implements the Time Zone Information Format (TZif) as specified in [RFC 8536](https://tools.ietf.org/html/rfc8536).
* [tzfile](tzfile): Parses the tzdb data and leapsecond files [distributed by IANA](https://data.iana.org/time-zones/) and [eggert/tz](https://github.com/eggert/tz).
* [ianatz](ianatz): Downloads and unpacks the IANA time zone database from https://data.iana.org/time-zones/.

## Resources
* [https://github.com/eggert/tz](https://github.com/eggert/tz): Time zone database and code
* [https://data.iana.org/time-zones](https://data.iana.org/time-zones): IANA file server for the time zone database

These HTML pages are part of [eggert/tz](https://github.com/eggert/tz) and conveniently hosted by IANA:

* [theory.html: Theory and pragmatics of the tz code and data](https://data.iana.org/time-zones/code/theory.html)
* [tz-art.html: Time and the Arts](https://data.iana.org/time-zones/code/tz-art.html)
* [tz-how-to.html: How to Read the tz Database Source Files](https://data.iana.org/time-zones/code/tz-how-to.html)

## License
Unless otherwise specified, all files in this repository are licensed under the [MIT license](LICENSE).

* `testdata/tzdata-2024b.tar.gz` is an unmodified copy of the tzdata release 2024b, [which is in the public domain](https://data.iana.org/time-zones/tzdb-2024b/LICENSE).
* `docs/zic.8.txt` is an unmodified copy of the zic man page from the tzcode distribution, also [in the public domain](https://data.iana.org/time-zones/tzdb-2024b/LICENSE).
* `docs/rfc8536.txt` is an unmodified distribution of [RFC 8536](https://tools.ietf.org/html/rfc8536) for documentation purposes.
* Package [tzif](tzif) implements the Time Zone Information Format (TZif) as specified in RFC 8536, and contains portions of the RFC for documentation purposes as well as samples from the RFC as part of the test suite.

I have made every effort to ensure that this project complies with applicable copyright laws and licensing terms. 
If you believe there are any issues or oversights regarding the licensing or use of any content in this repository, I would greatly appreciate it if you could open a GitHub issue to bring it to my attention.
