# navpatch

Navigate through a patch.

	go get github.com/tcard/navpatch

Go from this: https://github.com/tyba/typeschema/pull/10/files

... to this:

![Screenshot](https://cloud.githubusercontent.com/assets/727422/6881462/73db9eae-d561-11e4-9b2c-4f8eee1f8e49.png)

This command displays a patch, like the ones that `git diff` produces, in a typical filesystem navigator. The interface is served through a web browser.

# navpatch.serve

Serve `navpatch` visualizations for any Git repositories on demand.

	go get github.com/tcard/navpatch/navpatch.serve

This command launches a web server that, on demand, clones and manages Git repositories and displays the diffs between two commits. For example, after launching `navpatch.serve`:

	http://localhost:6177/github.com/tcard/navpatch?old=232eb53&new=6082eb0

That would clone the `git://git@github.com/tcard/navpatch` repo, It should work with any valid `git clone` URL.

## Dependencies

This command uses the `git` command, which should be installed in the system.


# To do

* A nicer CSS would be much appreciated.
* Docs!
* Tests?
* Overall integration with GitHub: 'See diff in GitHub', 'See pull request or commit', etc.
* Textual HTTP interface.
* See permissions changes.
* Detect renames.
* Navigate diffed lines pressing 'n' and 'p' or similar.
