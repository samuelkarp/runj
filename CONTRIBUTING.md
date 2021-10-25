## Open-source, with limited contribution

[Similar to SQLite](https://www.sqlite.org/copyright.html) and to
[Litestream](https://github.com/benbjohnson/litestream) (from whom I took
inspiration for this text), runj is open source but not yet fully open to code
contributions.

I am using this project to get a deeper understanding of FreeBSD, the OCI
specifications, and containerd.  I learn best by reading and doing, so I
prefer to write much of the code myself, at least for now.

I am grateful for community involvement, bug reports, pointers in what to learn,
and feature requests.  I do not wish to come off as anything but welcoming,
however, I've made the decision to keep this project more tightly held for my 
own learning at this point.

## How to contribute to runj

Thank you for your interest in contributing!  Because I'm using this project as
a mechanism for my own learning, I would really appreciate it if you can follow
these guidelines:

* Agree to license your contributions under the same terms as this repository,
  which can be found in [the LICENSE file](LICENSE).
* For anything except the most trivial one-line bugfixes or typo corrections,
  please **create an issue describing your change first**.  I want to learn
  about the problem that you'd like to address and how you'd like to do it
  first.
* Make sure the **tests pass** and **no new lint warnings** are introduced.
  Unit tests are written using Go's standard testing framework and can be run
  with `make test` (or `go test` if you're familiar with the Go tool).
  Integration tests consist of two Go binaries that cooperate inside and
  outside the jails and can be run using `make integ-test`.  For linting,
  runj uses `golangci-lint` (available in ports) which can be run with `make
  lint`.
* If possible and practical, please add new unit tests covering your changes.
* Git commit messages should contain a short summary of no more than 50
  characters in the first line and a longer description (when appropriate)
  wrapped at 72 characters on subsequent lines.
  [This guide](https://chris.beams.io/posts/git-commit/) is a good reference for
  writing good commit messages.

If you submit a pull request to me, please expect code review comments and
requests for changes to be made.  Depending on the nature of the comments and
how responsive you are to changes, I may do any of the following:

* Accept your change and add additional commits adjusting it to my preference
* Amend your change to incorporate my suggestions
* Re-write your change in its entirety

In any of these scenarios, I will credit you as either the author (in Git's
commit metadata) or co-author (in the Git commit message) of the change.

## Moving toward fully open-contribution

When I have reached my own goals for learning, I plan to open runj up for code
contribution.  In the meantime, please try runj out and let me know if you learn
something interesting yourself!
