# navpatch

Navigate through a patch.

	go get github.com/tcard/navpatch

Go from this: https://github.com/tyba/typeschema/pull/10/files

... to this:

![Screenshot](https://cloud.githubusercontent.com/assets/727422/6881462/73db9eae-d561-11e4-9b2c-4f8eee1f8e49.png)

This command displays a patch, like the ones that `git diff` produces, in a typical filesystem navigator. The interface is served through a web browser.

# To do

* A nicer CSS would be much appreciated.
* Tests?
* A server that automatically runs this command with GitHub URLs is underway.
* Overall integration with GitHub: 'See diff in GitHub', 'See pull request or commit', etc.
* Textual HTTP interface.
* See permissions changes.
* Detect renames.
