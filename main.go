package main

import (
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type FileInfo struct {
	Name     string
	Size     int64
	IsDir    bool
	URL      string
	Original string
	ModTime  string
	Parent   string
}

var tpl = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>目录列表</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            line-height: 1.6;
            margin: 20px;
        }
        h1 {
            color: #2c3e50;
        }
        .back-link {
            font-size: 14px;
            margin-bottom: 10px;
            display: inline-block;
            color: #2980b9;
            text-decoration: none;
        }
        .back-link:hover {
            text-decoration: underline;
        }
        ul {
            list-style-type: none;
            padding-left: 0;
        }
        li {
            margin: 8px 0;
            font-size: 16px;
        }
        .size {
            color: #7f8c8d;
            font-size: 14px;
            margin-left: 20px; /* 增加文件大小与链接之间的间距 */
        }
        .mod-time {
            color: #95a5a6;
            font-size: 14px;
        }
        .file, .directory {
            display: flex;
            align-items: center;
        }
        .file a, .directory a {
            margin-left: 8px;
            color: #34495e;
            text-decoration: none;
        }
        .file a:hover, .directory a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>

<h1>目录列表</h1>
<!-- 如果有上级目录，显示返回链接 -->
{{if .Parent}}
    <p><a href="{{.Parent}}" class="back-link">⬅ 返回上级</a></p>
{{end}}


<!-- 文件和目录列表 -->
<ul>
    {{range .Files}}
        <li class="{{if .IsDir}}directory{{else}}file{{end}}">
            <span class="icon">
                {{if .IsDir}}📁{{else}}📄{{end}}
            </span>
            <a href="{{.Original}}">{{.Name}}</a>
            
            <!-- 如果是文件，显示文件大小 -->
            {{if not .IsDir}}
                <span class="size" data-bytes="{{.Size}}">{{.Size}} bytes</span>
                <a href="{{.URL}}">下载</a>
            {{end}}
            
            <!-- 显示最后修改时间 -->
            <span class="mod-time"> &nbsp; {{.ModTime}}</span>
        </li>
    {{end}}
</ul>

</body>
<script>
  function humanSize(n) {
    const KB = 1024, MB = KB*1024, GB = MB*1024;
    if (n >= GB) return (n/GB).toFixed(2) + ' GB';
    if (n >= MB) return (n/MB).toFixed(2) + ' MB';
    if (n >= KB) return (n/KB).toFixed(2) + ' KB';
    return n + ' Byte';
  }
  document.querySelectorAll('.size').forEach(el => {
    const bytes = parseInt(el.getAttribute('data-bytes'), 10) || 0;
    el.textContent = humanSize(bytes);
  });
</script>
</html>
`

type PageData struct {
	Files  []FileInfo
	Parent string
}

func handler(w http.ResponseWriter, r *http.Request, root string) {
	//dir := "." + r.URL.Path
	//if root != "" {
	//	dir = root
	//}

	dir := root + r.URL.Path

	files, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var list []FileInfo
	for _, f := range files {
		info, _ := f.Info()
		name := f.Name()
		modTime := info.ModTime().Format("2006-01-02 15:04:05")
		var urlStr string
		var original string
		if f.IsDir() {
			urlStr = r.URL.Path + name + "/"
			original = r.URL.Path + name + "/"
		} else {
			encodedName := url.PathEscape(name)
			if r.URL.Path == "/" {
				urlStr = "/download/" + encodedName
				original = "/view/" + encodedName
			} else {
				urlStr = "/download" + r.URL.Path + encodedName
				original = "/view" + r.URL.Path + encodedName
			}
		}
		list = append(list, FileInfo{
			Name:     name,
			Size:     info.Size(),
			IsDir:    f.IsDir(),
			URL:      urlStr,
			Original: original,
			ModTime:  modTime,
		})
	}

	// 文件夹排前，名字排序
	sort.Slice(list, func(i, j int) bool {
		if list[i].IsDir != list[j].IsDir {
			return list[i].IsDir
		}
		return list[i].Name < list[j].Name
	})

	// 计算上级目录
	current := strings.TrimSuffix(r.URL.Path, "/")
	parent := ""
	if current != "" && current != "/" {
		parent = path.Dir(current) // 使用 path 包，永远 / 分隔
		if parent == "." || parent == "/" {
			parent = "/"
		} else {
			parent += "/" // 保证最后有 /
		}
	}

	t := template.Must(template.New("dir").Parse(tpl))
	t.Execute(w, PageData{Files: list, Parent: parent})
}

func downloadHandler(w http.ResponseWriter, r *http.Request, root string) {
	rawPath := r.URL.Path[len("/download"):] // 去掉 /download 前缀
	decodedPath, err := url.PathUnescape(rawPath)
	if err != nil {
		http.Error(w, "Invalid file name", http.StatusBadRequest)
		return
	}

	dir := root + decodedPath

	// filepath.Clean 函数用于清理路径字符串。它会规范化文件路径，去除路径中的冗余部分，比如多余的 . 和 .. 目录元素.
	filePath := filepath.Clean(dir)
	// os.Stat 函数用于获取指定文件或目录的状态信息（FileInfo）
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	log.Println(filePath)

	w.Header().Set("Content-Disposition", `attachment; filename="`+info.Name()+`"`)
	http.ServeFile(w, r, filePath)
}

func viewHandler(w http.ResponseWriter, r *http.Request, root string) {
	rawPath := r.URL.Path[len("/view"):]
	decodedPath, err := url.PathUnescape(rawPath)
	if err != nil {
		http.Error(w, "Invalid file name", http.StatusBadRequest)
		return
	}

	filePath := filepath.Clean(root + decodedPath)
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// 自动检测 MIME 类型
	f, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// 读取前 512 字节判断类型
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	contentType := http.DetectContentType(buf[:n])

	// 重置读取位置
	f.Seek(0, io.SeekStart)

	// 设置为 inline 显示
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `inline; filename="`+info.Name()+`"`)

	io.Copy(w, f)
}

/*
编译：
go build -o FileServer.exe goDemo2/Go-Download-Static-Files/version4

运行：
Go-Download-Static-Files
Go-Download-Static-Files -port=8081
Go-Download-Static-Files --port=8081
Go-Download-Static-Files --port=8081 -root="D:\temp\seata"
Go-Download-Static-Files --port=8081 --root="D:\temp\seata"

在 main3.go 上优化
获取指定目录下的文件，运行在指定端口，默认为 8080 端口
优化显示样式
访问：
http://127.0.0.1:8080
*/
func main() {
	// 定义命令行参数，默认值8080
	port := flag.String("port", "8080", "Server port")
	rootDir := flag.String("root", ".", "Root directory to serve files from")

	// 解析用户传入的命令行参数。如果用户没有提供该参数，会使用默认值。
	flag.Parse()

	addr := ":" + *port
	// 绝对路径
	absRoot, err := filepath.Abs(*rootDir)
	// 绝对路径，测试
	//absRoot, err = filepath.Abs("C:\\Users")

	absRoot = strings.ReplaceAll(absRoot, string(os.PathSeparator), "/")
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}
	log.Printf("Serving files from: %s\n", absRoot)

	// 文件下载处理
	http.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		downloadHandler(w, r, absRoot)
	})

	// 文件查看处理
	http.HandleFunc("/view/", func(w http.ResponseWriter, r *http.Request) {
		viewHandler(w, r, absRoot)
	})

	// 根目录文件处理
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, absRoot)
	})

	log.Printf("Serving on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
