package main

import (
	"net/http"
	"fmt"
	"path/filepath"
	"io/ioutil"
	"strings"
	"os"
	"mime"
	"strconv"
	"sort"
	"html/template"
)

const tpl = `<!doctype html>
<html>
<head>
    <meta charset="utf-8" />
    <title>Index of {{ .Path }}</title>

    <style>
        body {
            font-family:Arial;
            font-size:14px;

            margin:0px;
            padding:0px;
        }

        .top {
            background-color:#fff;
            border-bottom: 1px solid #d9d9da;
        }

        h1, table, .content p {
            width:1024px;
            margin:0 auto;
        }

        p {
            padding-bottom:5px;
        }

        h1 {
            padding:30px 0;
        }

        .content {
            background-color: #f7f7f7;
            padding:30px 0;
        }

        table {
            border-collapse: collapse;
        }

        thead {
            background-color:#f0f0f0;
        }

        th, td {
            padding:10px;
        }

        table, td {
            border:1px solid #d4d4d4;
        }

        tbody tr:nth-child(2n+1) {
            background-color: #fff;
        }
    </style>
</head>
<body>

<div class="top">
    <h1>Index of {{ .Path }}</h1>
</div>
<div class="content">

    <p>
        <a href="../">↑ Up</a>
    </p>

    <table>
        <thead>
        <tr>
            <th>Name</th>
            <th>Modified</th>
            <th>Size</th>
            <th>Mode</th>
        </tr>
        </thead>
        <tbody>
        {{ range .Entries }}
        <tr>
            <td>
                <a href="{{ .Name }}">{{ .Name }}</a>
            </td>
            <td>{{ .ModTime }}</td>
            <td>{{ .Size }}</td>
            <td>{{ .Mode }}</td>
        </tr>
        {{ else }}
        <tr>
            <td colspan="4">no entries</td>
        </tr>
        {{ end }}
        </tbody>
    </table>
</div>
</body>
</html>`

type AutoIndex struct {
	Root string
}

func (ai AutoIndex) handler(w http.ResponseWriter, r *http.Request) error {
	path := "/" + r.URL.Path[1:]

	resolvedPath := filepath.Join(ai.Root, filepath.FromSlash(filepath.Clean(path)))

	rel, err := filepath.Rel(ai.Root, resolvedPath)

	if err != nil {
		return err
	}

	/* check if file/dir exists */
	if fifo, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		return err
	} else if !fifo.IsDir() {
		/* it's a file, and it exists */
		return ai.readFile(w, resolvedPath)
	}

	/* make sure directories end in slash */
	if path[len(path) - 1:] != "/" {
		http.Redirect(w, r, path + "/", http.StatusFound)
		return nil
	}

	return ai.listFiles(w, resolvedPath, rel)
}

func (ai AutoIndex) readFile(w http.ResponseWriter, path string) error {
	bytes, err := ioutil.ReadFile(path)

	if err != nil {
		return err
	}

	mimeType := mime.TypeByExtension(path[strings.LastIndex(path, "."):])
	w.Header().Add("Content-Type", mimeType)
	w.Header().Add("Content-Length", strconv.Itoa(len(bytes)))

	w.Write(bytes)
	return nil
}

func (ai AutoIndex) listFiles(w http.ResponseWriter, path string, relativePath string) error {
	t, err := template.New("webpage").Parse(tpl)
	if err != nil {
		return err
	}

	entries, err := ReadDir(path)
	if err != nil {
		return err
	}

	data := struct {
		Path string
		Entries []os.FileInfo
	}{
		Path: relativePath,
		Entries: entries,
	}

	w.Header().Add("Content-Type", "text/html")
	err = t.Execute(w, data)

	if err != nil {
		return err
	}

	return nil
}

type httpErrorHandlerWrapper func(w http.ResponseWriter, r *http.Request) error

func (fn httpErrorHandlerWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := fn(w, r); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func ReadDir(dirname string) ([]os.FileInfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}

	sort.Slice(list, SortDirectoriesFirst(list))
	return list, nil
}

func SortDirectoriesFirst(fifos []os.FileInfo) func(i, j int) bool {
	// if either fifos[i] IsDir or fifos[j] IsDir, but not both,
	//   return true if fifos[i].IsDir(), false otherwise
	return func(i, j int) bool {
		// if golang had XOR, this could be simplified as:
		// if fifos[i].IsDir() XOR fifos[j].IsDir {
		//   return fifos[i].IsDir()
		// }
		if fifos[i].IsDir() && !fifos[j].IsDir() {
			return true
		}
		if !fifos[i].IsDir() && fifos[j].IsDir() {
			return false
		}
		return fifos[i].Name() < fifos[j].Name()
	}
}

func main() {
	root := os.Getenv("AUTOINDEX_ROOT")
	if root == "" {
		if len(os.Args) == 2 {
			root = os.Args[1]
		} else {
			cwd, err := os.Getwd()

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			root = cwd
		}
	}

	listenAddr := os.Getenv("AUTOINDEX_LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	autoIndex := AutoIndex{root}
	http.Handle("/", httpErrorHandlerWrapper(autoIndex.handler))
	panic(http.ListenAndServe(listenAddr, nil))
}
