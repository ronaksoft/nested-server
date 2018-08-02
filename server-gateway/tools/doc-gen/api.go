package main

import (
    "os"
    "log"
    "bufio"
    "fmt"
    "strings"
    "html/template"
    "path/filepath"
)

const jsonTemplate string = `
{
{{range $serviceIndex, $service := .Services}}{{if $serviceIndex}},{{end}}
"{{$service.Name}}":
	[{{range $methodIndex, $method := .Methods}}{{if $methodIndex}},{{end}}
		{
			"cmd": "{{$method.Command}}",
			"args": [
				{{range $argIndex, $arg := $method.Arguments}}{{if $argIndex}},{{end}}{
					"name": "{{.Name}}",
					"type": "{{.Type}}",
					"comment": "{{.Comment}}",
					"required": {{.Required}}
				}{{end}}
			],
			"pagination": {{$method.Pagination}}
		}{{end}}
	]{{end}}
}

`

type Line struct {
    Constructor string
    Params      []string
}

type ApiDoc struct {
    Services []Service
}

type Service struct {
    Name    string
    Methods []ServiceMethod
}

type ServiceMethod struct {
    Command    string
    Arguments  []Argument
    Pagination bool
}

type Argument struct {
    Name     string
    Type     string
    Comment  string
    Required bool
}

func main() {
    apiDoc := ApiDoc{}
    filepath.Walk("./", func(path string, info os.FileInfo, err error) error {
        if info.IsDir() {
            fmt.Println(info.Name(), path)
            if _, err := os.Stat(fmt.Sprintf("%s/functions.go", path)); !os.IsNotExist(err) {
                apiDoc.Services = append(apiDoc.Services, ProduceJSON(info.Name(), fmt.Sprintf("%s/functions.go", path)))
                ExecuteTemplate("api-doc.json", apiDoc)
            }

        }
        return nil
    })

}

func ScanComments(path string) []Line {
    var scanner *bufio.Scanner
    var lines []Line
    if file, err := os.Open(path); err != nil {
        log.Fatal(err)
    } else {
        defer file.Close()
        scanner = bufio.NewScanner(file)
    }

    for scanner.Scan() {
        var fields []string
        txt := scanner.Text()

        // Only for comments
        if strings.HasPrefix(txt, "//") {
            fields = strings.Fields(txt)
            if len(fields) > 2 {
                line := Line{
                    Constructor: strings.TrimRight(fields[1], ": "),
                }
                if len(fields) > 6 {
                    line.Params = append(line.Params, fields[2], fields[3], fields[4], strings.Join(fields[5:], " "))
                } else {
                    line.Params = append(line.Params, fields[2:]...)
                }
                lines = append(lines, line)
            }
        }
    }
    return lines
}

func ExecuteTemplate(filename string, apis ApiDoc) {
    t, _ := template.New(filename).Parse(jsonTemplate)
    if f, err := os.OpenFile(fmt.Sprintf("%s", filename), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0750); err != nil {
        log.Println("ExecuteTemplate::", err.Error())
    } else {
        t.Execute(f, apis)
    }
}

func ProduceJSON(serviceName, path string) Service {
    lines := ScanComments(path)
    apis := Service{
        Name: serviceName,
    }
    idx := 0
    var m ServiceMethod
    for {
        switch lines[idx].Constructor {
        case "@Command":
            if len(m.Command) > 0 {
                apis.Methods = append(apis.Methods, m)
            }
            m = ServiceMethod{
                Command: lines[idx].Params[0],
            }
        case "@Input":
            argument := Argument{
                Name: lines[idx].Params[0],
                Type: lines[idx].Params[1],
            }
            if len(lines[idx].Params) > 2 {
                if lines[idx].Params[2] == "*" {
                    argument.Required = true
                }
            }
            if len(lines[idx].Params) == 4 {
                argument.Comment = lines[idx].Params[3]
            }
            m.Arguments = append(m.Arguments, argument)
        case "@Pagination":
            m.Pagination = true
        case "@CommandInfo":
        default:
            log.Println("ERROR::", lines[idx])
        }
        idx++
        if idx >= len(lines) {
            break
        }
    }
    apis.Methods = append(apis.Methods, m)
    return apis
}
