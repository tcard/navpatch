# navpatch

Navigate through a patch.

	go get github.com/tcard/navpatch

Go from this: https://github.com/tyba/typeschema/pull/10/files

... to this:

![Screenshot](https://cloud.githubusercontent.com/assets/727422/6881462/73db9eae-d561-11e4-9b2c-4f8eee1f8e49.png)

([See live demo](#demo).)

This command displays a patch, like the ones that `git diff` produces, in a typical filesystem navigator. The interface is served through a web browser.

# navpatch.serve

Serve `navpatch` visualizations for any Git repositories on demand.

	go get github.com/tcard/navpatch/navpatch.serve

This command launches a web server that, on demand, clones and manages Git repositories and displays the diffs between two commits. For example, after launching `navpatch.serve`, going to...

	http://localhost:6177/github.com/tcard/navpatch?old=232eb53&new=6082eb0

... would clone the `git+ssh://git@github.com/tcard/navpatch` repo, copy it in another directory for this session, and display the diff between commits `232eb53` and `6082eb0`.

It should work with any valid `git clone` URL. It also recognizes Github pull request URLs:

	http://localhost:6177/github.com/tyba/typeschema/pull/10

... althought that assumes pull requests against the main branch not yet merged. (See [to do list](#to-do).)

For full options, including whitelisting and concurrent sessions limiting, see:

	navpatch.serve -h

## Demo

You can check a live demo running at [http://fanyare.tcardenas.me:6177](http://fanyare.tcardenas.me:6177).

Only repos at `github.com/tyba` and `github.com/tcard` will work, though.

Some demo URLs:

* http://fanyare.tcardenas.me:6177/github.com/tyba/typeschema?old=29b179c&new=52f9f11
* http://fanyare.tcardenas.me:6177/github.com/tcard/navpatch?old=232eb53&new=6082eb0
* http://fanyare.tcardenas.me:6177/github.com/tcard/navpatch/pull/1

## Dependencies

This command uses the `git` command, which should be installed in the system.

# To do

* A nicer CSS would be much appreciated.
* Docs!
* Tests?
* Overall integration with GitHub: 'See diff in GitHub', 'See pull request or commit', etc.
* Scrape Github HTML instead of using the `git` command; this would:
  - Improve performance, and reduce headache with caches, etc.
  - Remove dependency on the `git` command for Github repos.
  - Give better support for pull requests. From the Github HTML we can Scrape which branch the pull request is to be merged on.
* Textual HTTP interface.
* See permissions changes.
* Detect renames.
* Navigate diffed lines pressing 'n' and 'p' or similar.
