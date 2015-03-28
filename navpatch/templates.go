package navpatch

import (
	"fmt"
	"html"
	"html/template"
	"strconv"
	"strings"
)

type tplFullData struct {
	Title    string
	TreeData tplTreeData
	Nav      *Navigator
}

type tplTreeData struct {
	Levels      []tplTreeDataLevel
	Nav         *Navigator
	LinksPrefix string
}

type tplTreeDataLevel struct {
	Path    string
	Entries []tplTreeDataLevelEntry
	Body    string
	Error   error
}

type tplTreeDataLevelEntry struct {
	DiffStats
	Name   string
	IsDir  bool
	IsOpen bool
}

var templates = template.Must(template.New("").Funcs(template.FuncMap{
	"concat": func(s ...string) string {
		return strings.Join(s, "")
	},
	"marginLeft": func(lvl int) template.HTMLAttr {
		return template.HTMLAttr(strconv.Itoa(lvl * 200))
	},
	"toString": func(bs []byte) string {
		return string(bs)
	},
	"colorify": func(diff string) template.HTML {
		ret := `<table class="diff"><tbody>`
		lines := strings.Split(diff, "\n")
		for i, line := range lines {
			if len(line) < 2 {
				continue
			}
			class := ""
			sign, line := line[:2], line[2:]
			if sign == "+ " {
				class = "addition"
			} else if sign == "- " {
				class = "deletion"
			}

			ret += fmt.Sprintf(`<tr class="%s">`, class)
			ret += fmt.Sprintf(`<td class="line-num">%d</td>`, i)
			ret += fmt.Sprintf(`<td class="line-content %s">%s</td>`, class, html.EscapeString(line))
			ret += "</tr>"
		}
		ret += `</tbody></table>`
		return template.HTML(ret)
	},
}).Parse(`
{{define "full"}}
<!DOCTYPE html>
<html style="margin: 0; padding: 0; height: 100%;">
<head>
  <meta charset="utf-8">
  <title>{{.Title}}</title>
  <style>
  body {
  	font-family: "Helvetica", sans-serif;
  }

  a.file-link {
  	display: block;
  	margin: 0;
  	padding: 5px 10px 5px 10px;
    font-size: small;
  	border-bottom: 1px solid #eee;
  	color: #333;
  	width: 179px;
  	text-decoration: none;
  }

  a.file-link:hover {
  	background-color: #f3f3f3;
  }

  a.file-link.active:hover {
  	background-color: #0ae;
  }

  a.file-link.active {
  	color: white;
  	background-color: #0bf;
  }

  a.file-link .link-right {
  	float: right;
  }

  .dir-arrow {
  	font-size: xx-small;
  	vertical-align: middle;
  }

  .additions {
  	font-size: small;
  	color: #408840;
  	font-weight: bold;
  	vertical-align: middle;
  }

  .active .additions, .active .deletions {
  	color: white;
  }

  .deletions {
  	font-size: small;
  	color: #884040;
  	font-weight: bold;
  	vertical-align: middle;
  }

  table.diff {
  	display: block;
    border-collapse: collapse;
  	font-family: monospace;
  	width: 800px;
  	height: 100%;
  	overflow-y: scroll;
  	overflow-x: scroll;
  }

  table.diff .line-num {
  	width: 20px;
  	text-align: right;
  	color: #aaa;
  }

  table.diff .line-content {
    white-space: pre;
  }

  table.diff .addition {
  	background-color: rgb(219, 255, 219);
  }

  table.diff .deletion {
  	background-color: rgb(255, 219, 219);;
  }

  div.folder {
    position: absolute;
    top: 0;
    height: 100%;
    overflow-x: hidden;
    overflow-y: scroll;
    border-right: 1px solid #aaa;
  }

  .error {
    padding: 10px;
    background-color: #faa;
    color: #600;
    height: auto;
    max-weight: 100%;
    min-weight: 100%;
    width: 800px;
  }
  </style>
</head>

<body style="margin: 0; padding: 0; height: 100%;">
  {{template "tree" .TreeData}}

  <script type="text/javascript">
  window.scrollTo(document.body.offsetWidth - 200, 0);
  </script>
</body>
</html>
{{end}}

{{define "tree"}}
{{range $i, $level := .Levels}}
	<div class="folder" style="left: {{marginLeft $i}}px;">

  {{with .Error}}
    <div class="error">
      <p>Error:</p>

      <pre>{{.}}</pre>

      <p>This typically means that the provided patch wasn't supposed to be
      applied to the provided base directory.</p>

      <p>Base directory:</p>

      <pre>{{$.Nav.BasePath}}</pre>

      <p>Patch:</p>

      <pre>{{toString $.Nav.RawPatch}}</pre>
    </div>
	{{else}}{{with .Body}}
		{{colorify .}}
	{{else}}
		{{range .Entries}}
			<a class="file-link {{with .IsOpen}}active{{end}}" href="{{concat $.LinksPrefix $level.Path "/" .Name }}">
				<span class="link-name">{{.Name}}</span>
				<span class="link-right">
				{{with .Additions}}<span class="additions">+{{.}}</span>{{end}}
				{{with .Deletions}}<span class="deletions">-{{.}}</span>{{end}}
				{{with .IsDir}}<span class="dir-arrow">▶</span>{{end}}
				</span>
			</a>
		{{end}}
	{{end}}{{end}}

	</div>
{{end}}
{{end}}
`))
