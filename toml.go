package toml

import (
	"bytes"
	"strings"
	"errors"
	"unicode"
	"strconv"
	"fmt"
	"time"
	l "log"
)

func log(args ... interface{}) {
	if !DEBUG {
		return
	}
	l.Println(args...)
}

var DEBUG = false

type Type int

const (
	Float64        Type = iota
	Int
	String
	Boolean
	Time
	StringArray
	IntArray
	InterfaceArray
)

func (t *Toml) Combine(y *Toml) {
	yTree := y.root
	tTree := t.root
	combine(tTree, yTree)
}
func combine(tTree *Tree, yTree *Tree) {
	for k, v := range yTree.Nodes {
		tTree.Nodes[k] = v
	}
	for k, yChild := range yTree.children {
		if tChild, ok := tTree.children[k]; ok {
			combine(tChild, yChild)
		} else {
			tTree.children[k] = yChild
		}
	}
}
func Parse(src []byte) (*Toml, error) {
	toml := &Toml{}
	toml.root = NewTree("", nil)
	tree := toml.root
	lines := bytes.Split(src, []byte{'\n'})
	for i := 0; i < len(lines); i++ {
		v := lines[i]
		v = bytes.TrimSpace(v)
		log(i, string(v))
		if len(v) == 0 {
			log(i, "# EMPTY")
			continue
		}
		if v[0] == '[' {
			lastPos := bytes.IndexByte(v, ']')
			if lastPos == -1 {
				return nil, errors.New("格式错误 " + string(v))
			}
			name := v[1:lastPos]
			pos := bytes.IndexByte(name, '.')
			if pos == -1 {
				child := NewTree(string(name), tree)
				toml.root.children[string(name)] = child
				tree = child
			} else {
				for {
					log("tree.Name", tree.parent, tree.Name, string(name))
					if strings.HasPrefix(string(name), tree.Name+".") {
						child := NewTree(string(name), tree)
						tree.children[string(name)[len(tree.Name)+1:]] = child
						tree = child
						break
					} else {
						tree = tree.parent
						if tree == toml.root {
							return nil, errors.New(fmt.Sprint("错误的格式 在第", i, "行"))
						}
					}
				}
			}
		} else {
			equalPos := bytes.Index(v, []byte{'='})
			if equalPos == -1 {
				continue
			}
			var node G
			name := bytes.TrimSpace(v[:equalPos])
			value := bytes.TrimLeftFunc(v[equalPos+1:], unicode.IsSpace)
			if bytes.HasPrefix(value, []byte(`"""`)) {
				stringValue := ""
				if len(value) > 6 && bytes.HasSuffix(value, []byte(`"""`)) {
					stringValue = string(value[3:len(value)-3])
				} else {
					bf := bytes.NewBuffer(value[3:])
					for {
						i++
						text := lines[i]
						if bytes.HasSuffix(bytes.TrimSpace(text), []byte{'"', '"', '"'}) {
							text = bytes.TrimRightFunc(text, unicode.IsSpace)
							text = text[:len(text)-3]
							bf.Write(text)
							break
						} else {
							bf.Write(text)
						}
					}
					stringValue = bf.String()
				}

				node = KVNode{
					Node: Node{
						Name:  string(name),
						Maybe: String,
					},
					Value:    stringValue,
					MayValue: stringValue,
				}
			} else if value[0] == '"' {
				lastIndex := -1
				for j := 0; j < len(value); j++ {
					if value[j] == '\\' {
						j++
						continue
					}
					if value[j] == '"' {
						lastIndex = j + 1
					}
				}

				stringValue, err := strconv.Unquote(string(value[:lastIndex]))
				if err != nil {
					return nil, err
				}
				node = KVNode{
					Node: Node{
						Name:  string(name),
						Maybe: String,
					},
					Value:    stringValue,
					MayValue: stringValue,
				}
			} else if bytes.Equal(value[:4], []byte("true")) || bytes.Equal(value[:4], []byte("false")) {
				boolean := bytes.Equal(value[:4], []byte("true"))
				node = KVNode{
					Node: Node{
						Name:  string(name),
						Maybe: Boolean,
					},
					Value:    string(value[:4]),
					MayValue: boolean,
				}
			} else if value[0] == '[' {
				value = append(value, '\n')
				arr, er := ReadArray(value, func() ([]byte, bool) {
					i++
					if i == len(lines) {
						return nil, false
					}
					return append(lines[i], '\n'), true
				})
				if er != nil {
					return nil, er
				}
				node = ArrayNode{
					Node: Node{
						Name:  string(name),
						Maybe: InterfaceArray,
					},
					Value: arr,
				}
			} else {
				commetpos := bytes.IndexByte(value, '#')
				if commetpos != -1 {
					value = value[:commetpos]
				}
				value = bytes.TrimRightFunc(value, unicode.IsSpace)
				intValue, err := strconv.Atoi(string(value))
				if err == nil {
					node = KVNode{
						Node: Node{
							Name:  string(name),
							Maybe: Int,
						},
						Value:    string(value),
						MayValue: intValue,
					}
				} else {
					floatValue, err := strconv.ParseFloat(string(value), 64)
					if err == nil {
						node = KVNode{
							Node: Node{
								Name:  string(name),
								Maybe: Float64,
							},
							Value:    string(value),
							MayValue: floatValue,
						}
					} else {
						t, err := time.Parse(time.RFC3339, string(value))
						if err == nil {
							node = KVNode{
								Node: Node{
									Name:  string(name),
									Maybe: Time,
								},
								Value:    string(value),
								MayValue: t,
							}
						}
						if err != nil {
							return nil, errors.New(fmt.Sprint("不能识别的 ", i, "行"))
						}
					}
				}
			}
			tree.Nodes[string(name)] = node
		}
	}
	return toml, nil
}

func ReadArray(first []byte, next func() ([]byte, bool)) ([]G, error) {
	i := 0
	return readArray(func() (byte, bool) {
		for i == len(first) {
			var has bool
			first, has = next()
			if !has {
				return ' ', false
			}
			i = 0
		}
		b := first[i]
		i++
		return b, true
	})
}
func readArray(next func() (byte, bool)) (arr []G, err error) {
	b, has := next()
	if !has {
		return nil, errors.New("toml文件格式错误")
	}
	if b != '[' {
		return nil, errors.New("toml文件格式错误")
	}
	arr = []G{}
	for {
		b, has := next()
		if !has {
			return nil, errors.New("toml文件格式错误")
		}
		switch b {
		case '#':
			for {
				b, has := next()
				if !has {
					return nil, errors.New("toml文件格式错误")
				}
				if b == '\n' {
					break
				}
			}
		case ' ', ',', '\n', '\t':
		// nothing to do
		case ']':
			return arr, nil
		case '"':
			str, err := readString(next)
			if err != nil {
				return nil, err
			}
			arr = append(arr, str)
		default:
			buf := bytes.NewBuffer([]byte{b})
			for {
				char, has := next()
				if !has {
					return nil, errors.New("1.文件结尾" + buf.String())
				}

				if char == ',' || char == ' ' || char == '\n' || char == '\t' {
					break
				}
				buf.WriteByte(char)
			}

			number := strings.TrimSpace(buf.String())
			if len(number) == 0 {
				continue
			}
			log("NUMBER", number)
			ivalue, err := strconv.Atoi(number)
			if err == nil {
				arr = append(arr, ivalue)
			} else if err != nil {
				fvalue, err := strconv.ParseFloat(number, 64)
				if err == nil {
					arr = append(arr, fvalue)
				} else {
					return nil, errors.New(fmt.Sprint("不支持啊 老铁", number))
				}
			}

		}
	}

}
func readString(next func() (byte, bool)) (string, error) {
	buf := bytes.NewBuffer([]byte{'"'})
	for {
		b, has := next()
		if !has {
			return "", errors.New("2.文件结尾")
		}
		buf.WriteByte(b)
		if b == '\\' {
			nextChar, has := next()
			if !has {
				return "", errors.New("3.文件结尾")
			}
			buf.WriteByte(nextChar)
		}
		if b == '\n' {
			return "", errors.New("字符串不能跨行")
		}
		if b == '"' {
			return strconv.Unquote(buf.String())
		}
	}
}

type Toml struct {
	root *Tree
}

func (t *Toml) Get(path string) (interface{}) {
	tree := t.root
	pathes := strings.Split(path, ".")
	for i, v := range pathes {
		child, ok := tree.children[v]
		if ok {
			log("FIND CHILD", v)
			tree = child
		} else {
			value, ok := tree.Nodes[v]
			if !ok {
				return nil
			}
			log("HAS VALUE", v)
			switch t := value.(type) {
			case KVNode:
				return t.MayValue
			case ArrayNode:
				if len(pathes)-1 != i {
					number := pathes[i+1]
					num, err := strconv.Atoi(number)
					if err != nil {
						return nil
					}
					return t.Value[num]
				}
				return t.Value
			}
		}
	}

	return nil
}

type G interface{}
type Tree struct {
	Name     string
	Nodes    map[string]G
	children map[string]*Tree
	parent   *Tree
}

func NewTree(name string, parent *Tree) (*Tree) {
	return &Tree{
		Name:     name,
		children: map[string]*Tree{},
		Nodes:    map[string]G{},
		parent:   parent,
	}
}

type Node struct {
	Name  string
	Maybe Type
}
type KVNode struct {
	Node
	Value    string
	MayValue G
}
type ArrayNode struct {
	Node
	Value []G
}
