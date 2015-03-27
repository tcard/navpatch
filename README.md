# navpatch

Navigate through a patch.

	go get github.com/tcard/navpatch

Go from this: https://github.com/tyba/typeschema/pull/10

To this:

![Screenshot](https://s3.amazonaws.com/f.cl.ly/items/3b161u0529081L170j3l/Captura%20de%20pantalla%202015-03-27%20a%20las%2019.43.55.png)

This command displays a patch, like the ones that `git diff` produces, in a typical filesystem navigator. The interface is served through a web browser.

# To do

* A nicer CSS would be much appreciated.
* Tests?
* A server that automatically runs this command with GitHub URLs is underway.
* Overall integration with GitHub: 'See diff in GitHub', 'See pull request or commit', etc.
* Better separation of concers to use as a library.
* Textual HTTP interface.
* See mode changes.
* Detect renames.
