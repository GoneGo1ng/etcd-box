package main

import (
	"context"
	jsoniter "github.com/json-iterator/go"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"go.etcd.io/etcd/clientv3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

type Node struct {
	name      string
	key       string
	icon      string
	parent    *Node
	children  []*Node
	rootName  string
	connected bool
}

func newNode(name, key, icon, rootName string, parent *Node) *Node {
	return &Node{
		name:     name,
		key:      key,
		icon:     icon,
		parent:   parent,
		rootName: rootName,
	}
}

var _ walk.TreeItem = new(Node)

func (d *Node) Text() string {
	return d.name
}

func (d *Node) Parent() walk.TreeItem {
	if d.parent == nil {
		return nil
	}
	return d.parent
}

func (d *Node) ChildCount() int {
	if d.children == nil {
		return 0
	}
	return len(d.children)
}

func (d *Node) ChildAt(index int) walk.TreeItem {
	return d.children[index]
}

func (d *Node) Image() interface{} {
	if d.icon != "" {
		icon, err := walk.NewIconFromFile(d.icon)
		if err != nil {
			zap.L().Error("Error", zap.Error(err))
			return nil
		}
		return icon
	}
	return nil
}

func (d *Node) addNode(keys []string, key, rootName string) {
	if len(keys) == 0 {
		return
	}
	var child *Node
	if d.children != nil {
		for _, v := range d.children {
			if v.name == keys[0] {
				child = v
			}
		}
	}
	if child == nil {
		icon := "img/dir.ico"
		if len(keys) == 1 {
			icon = "img/file.ico"
		}
		child = newNode(keys[0], key, icon, rootName, d)
		d.children = append(d.children, child)
	}
	child.addNode(keys[1:], key, rootName)
}

func (d *Node) refreshNodeIcon(icon string) {
	d.icon = icon
	treeModel.PublishItemChanged(d)
}

type NodeTreeModel struct {
	walk.TreeModelBase
	roots []*Node
}

func newNodeTreeModel() (*NodeTreeModel, error) {
	model := new(NodeTreeModel)

	root := newNode("All", "", "img/menu.ico", "", nil)
	model.roots = append(model.roots, root)
	for _, ec := range etcdConfigs {
		root.children = append(root.children,
			newNode(ec.Name, "", "img/unconnected.ico", "", root))
	}

	return model, nil
}

var _ walk.TreeModel = new(NodeTreeModel)

func (*NodeTreeModel) LazyPopulation() bool {
	return true
}

func (m *NodeTreeModel) RootCount() int {
	return len(m.roots)
}

func (m *NodeTreeModel) RootAt(index int) walk.TreeItem {
	return m.roots[index]
}

type editRequiredValidator struct {
}

var editRequiredValidatorSingleton walk.Validator = editRequiredValidator{}

func EditRequiredValidator() walk.Validator {
	return editRequiredValidatorSingleton
}

func (editRequiredValidator) Validate(v interface{}) error {
	if v == nil || v.(string) == "" {
		// For Widgets like ComboBox nil is passed to indicate "no selection".
		return walk.NewValidationError("Required", "Please enter a string.")
	}

	return nil
}

type EditRequired struct {
}

func (EditRequired) Create() (walk.Validator, error) {
	return EditRequiredValidator(), nil
}

type numberRequiredValidator struct {
}

var numberRequiredValidatorSingleton walk.Validator = numberRequiredValidator{}

func NumberRequiredValidator() walk.Validator {
	return numberRequiredValidatorSingleton
}

func (numberRequiredValidator) Validate(v interface{}) error {
	if v == nil || v.(float64) <= 0 {
		// For Widgets like ComboBox nil is passed to indicate "no selection".
		return walk.NewValidationError("Required", "Please enter a number greater than zero.")
	}

	return nil
}

type NumberRequired struct {
}

func (NumberRequired) Create() (walk.Validator, error) {
	return NumberRequiredValidator(), nil
}

type EtcdConfig struct {
	Name     string
	Endpoint string
	Host     string
	Port     float64
	Username string
	Password string
	Client   *clientv3.Client `json:"-"`
}

var etcdConfigs map[string]*EtcdConfig

var waitDlg *walk.Dialog

func createWaitDialog() {
	var cancelPB *walk.PushButton

	if err := (Dialog{
		AssignTo:     &waitDlg,
		Title:        "Connecting...",
		CancelButton: &cancelPB,
		MinSize:      Size{250, 150},
		Layout:       VBox{},
		Children: []Widget{
			ProgressBar{
				MarqueeMode: true,
			},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo:  &cancelPB,
						Text:      "Cancel",
						OnClicked: func() { waitDlg.Cancel() },
					},
					HSpacer{},
				},
			},
		},
	}.Create(mainWindow)); err != nil {
		zap.L().Error("Error", zap.Error(err))
		panic(err)
	}
}

type Search struct {
	Key string
}

var searchKey string

func createSearchDialog() *walk.Dialog {
	var db *walk.DataBinder
	var dlg *walk.Dialog
	var searchPB, cancelPB *walk.PushButton

	search := new(Search)

	if err := (Dialog{
		AssignTo:      &dlg,
		Title:         "Search",
		DefaultButton: &searchPB,
		CancelButton:  &cancelPB,
		MinSize:       Size{350, 150},
		DataBinder: DataBinder{
			AssignTo:       &db,
			DataSource:     search,
			ErrorPresenter: ToolTipErrorPresenter{},
		},
		Layout: VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{Text: "Key:"},
					LineEdit{Text: Bind("Key")},
				},
			},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &searchPB,
						Text:     "Search",
						OnClicked: func() {
							if err := db.Submit(); err != nil {
								zap.L().Error("Error", zap.Error(err))
								walk.MsgBox(mainWindow, "Error", "Search failed.", walk.MsgBoxIconError)
								return
							}
							searchKey = search.Key
							dlg.Accept()
						},
					},
					PushButton{
						AssignTo:  &cancelPB,
						Text:      "Cancel",
						OnClicked: func() { dlg.Cancel() },
					},
					HSpacer{},
				},
			},
		},
	}.Create(mainWindow)); err != nil {
		zap.L().Error("Error", zap.Error(err))
		panic(err)
	}

	return dlg
}

func createAddNewConnectionDialog() *walk.Dialog {
	var db *walk.DataBinder
	var dlg *walk.Dialog
	var savePB, cancelPB *walk.PushButton

	ec := new(EtcdConfig)
	ec.Name = "ConnectionName"
	ec.Host = "127.0.0.1"
	ec.Port = 2379

	if err := (Dialog{
		AssignTo:      &dlg,
		Title:         "Add New Connection",
		DefaultButton: &savePB,
		CancelButton:  &cancelPB,
		MinSize:       Size{350, 200},
		DataBinder: DataBinder{
			AssignTo:       &db,
			DataSource:     ec,
			ErrorPresenter: ToolTipErrorPresenter{},
		},
		Layout: VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{Text: "Name:"},
					LineEdit{Text: Bind("Name", EditRequired{})},
					Label{Text: "Address:"},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							LineEdit{Text: Bind("Host", EditRequired{})},
							Label{Text: ":"},
							NumberEdit{
								Value:              Bind("Port", NumberRequired{}),
								SpinButtonsVisible: true,
								MinSize:            Size{Width: 80},
							},
						},
					},
					Label{Text: "Username:"},
					LineEdit{Text: Bind("Username")},
					Label{Text: "Password:"},
					LineEdit{PasswordMode: true, Text: Bind("Password")},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &savePB,
						Text:     "Save",
						OnClicked: func() {
							if err := db.Submit(); err != nil {
								zap.L().Error("Error", zap.Error(err))
								walk.MsgBox(mainWindow, "Error", "Save failed.", walk.MsgBoxIconError)
								return
							}

							if etcdConfigs[ec.Name] != nil {
								walk.MsgBox(mainWindow, "Warning", "Please enter a name that does not exist.", walk.MsgBoxIconWarning)
								return
							}

							ec.Endpoint = "http://" + ec.Host + ":" + strconv.FormatFloat(ec.Port, 'f', -1, 64)
							etcdConfigs[ec.Name] = ec

							content, err := jsoniter.Marshal(etcdConfigs)
							if err != nil {
								zap.L().Error("Error", zap.Error(err))
								walk.MsgBox(mainWindow, "Error", "Save failed.", walk.MsgBoxIconError)
								return
							}
							if err := ioutil.WriteFile("config.json", content, 0777); err != nil {
								zap.L().Error("Error", zap.Error(err))
								walk.MsgBox(mainWindow, "Error", "Save failed.", walk.MsgBoxIconError)
								return
							}

							treeModel.PublishItemInserted(newNode(ec.Name, "", "img/unconnected.ico", "", treeModel.roots[0]))

							dlg.Accept()
						},
					},
					PushButton{
						AssignTo:  &cancelPB,
						Text:      "Cancel",
						OnClicked: func() { dlg.Cancel() },
					},
					HSpacer{},
				},
			},
		},
	}.Create(mainWindow)); err != nil {
		zap.L().Error("Error", zap.Error(err))
		panic(err)
	}

	return dlg
}

var mousePosition *MousePosition

type MousePosition struct {
	x int
	y int
}

func (mp *MousePosition) resetMousePosition(x, y int) {
	mp.x = x
	mp.y = y
}

var mainWindow *walk.MainWindow
var nodes *walk.TreeView
var key *walk.TextEdit
var value *walk.TextEdit
var splitter *walk.Splitter
var showConnectAction *walk.Action
var showReconnectAction *walk.Action
var showDisconnectAction *walk.Action
var showSearchAction *walk.Action
var showDeleteAction *walk.Action

var treeModel *NodeTreeModel

func main() {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	cfg.OutputPaths = []string{"ETCDBox.log"}
	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)

	mousePosition = &MousePosition{0, 0}

	etcdConfigs = make(map[string]*EtcdConfig)
	f, err := ioutil.ReadFile("config.json")
	if err != nil {
		zap.L().Error("Error", zap.Error(err))
		panic(err)
	}
	if err := jsoniter.UnmarshalFromString(string(f), &etcdConfigs); err != nil {
		zap.L().Error("Error", zap.Error(err))
		panic(err)
	}

	treeModel, err = newNodeTreeModel()
	if err != nil {
		zap.L().Error("Error", zap.Error(err))
		panic(err)
	}

	if err := (MainWindow{
		AssignTo: &mainWindow,
		Title:    "ETCD Box",
		MinSize:  Size{600, 400},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					PushButton{
						Text: " Add New Connection ",
						OnClicked: func() {
							createAddNewConnectionDialog().Run()
						},
					},
					HSpacer{},
				},
			},
			HSplitter{
				AssignTo: &splitter,
				Children: []Widget{
					TreeView{
						AssignTo:      &nodes,
						Model:         treeModel,
						ItemHeight:    20,
						StretchFactor: 1,
						ContextMenuItems: []MenuItem{
							Action{
								AssignTo: &showConnectAction,
								Text:     "Connect",
								OnTriggered: func() {
									if mousePosition != nil {
										if item := nodes.ItemAt(mousePosition.x, mousePosition.y); item != nil {
											node := item.(*Node)
											connect(node)
										}
									}
								},
							},
							Action{
								AssignTo: &showReconnectAction,
								Text:     "Reconnect",
								OnTriggered: func() {
									if mousePosition != nil {
										if item := nodes.ItemAt(mousePosition.x, mousePosition.y); item != nil {
											node := item.(*Node)
											disconnect(node)
										}
										if item := nodes.ItemAt(mousePosition.x, mousePosition.y); item != nil {
											node := item.(*Node)
											connect(node)
										}
									}
								},
							},
							Action{
								AssignTo: &showDisconnectAction,
								Text:     "Disconnect",
								OnTriggered: func() {
									if mousePosition != nil {
										if item := nodes.ItemAt(mousePosition.x, mousePosition.y); item != nil {
											node := item.(*Node)
											disconnect(node)
										}
									}
								},
							},
							Action{
								AssignTo: &showSearchAction,
								Text:     "Search",
								OnTriggered: func() {
									if mousePosition != nil {
										if item := nodes.ItemAt(mousePosition.x, mousePosition.y); item != nil {
											node := item.(*Node)
											if createSearchDialog().Run() == walk.DlgCmdOK {
												search(searchKey, node)
											}
										}
									}
								},
							},
							Action{
								AssignTo: &showDeleteAction,
								Text:     "Delete",
								OnTriggered: func() {
									if mousePosition != nil {
										if item := nodes.ItemAt(mousePosition.x, mousePosition.y); item != nil {
											node := item.(*Node)
											delete(etcdConfigs, node.name)
											content, err := jsoniter.Marshal(etcdConfigs)
											if err != nil {
												zap.L().Error("Error", zap.Error(err))
												walk.MsgBox(mainWindow, "Error", "Delete failed.", walk.MsgBoxIconError)
												return
											}
											if err := ioutil.WriteFile("config.json", content, 0777); err != nil {
												zap.L().Error("Error", zap.Error(err))
												walk.MsgBox(mainWindow, "Error", "Delete failed.", walk.MsgBoxIconError)
												return
											}

											disconnect(node)
											treeModel.PublishItemRemoved(node)
										}
									}
								},
							},
						},
						OnMouseDown: func(x, y int, button walk.MouseButton) {
							if button == walk.LeftButton {
								if item := nodes.ItemAt(x, y); item != nil {
									node := item.(*Node)
									if node.parent != nil && node.key == "" && !node.connected {
										connect(node)
									} else if node.children == nil && node.key != "" {
										if ec := etcdConfigs[node.rootName]; ec != nil && ec.Client != nil {
											client := ec.Client
											resp, err := client.Get(context.Background(), node.key)
											if err != nil {
												zap.L().Error("Error", zap.Error(err))
												walk.MsgBox(mainWindow, "Error", "Connect failed.", walk.MsgBoxIconError)
												return
											}
											if resp != nil && resp.Kvs != nil {
												for _, v := range resp.Kvs {
													key.SetText(node.key)
													value.SetText(string(v.Value))
												}
											}
										} else {
											// TODO
										}
									} else {
										key.SetText("")
										value.SetText("")
									}
								}
							} else if button == walk.RightButton {
								if item := nodes.ItemAt(x, y); item != nil {
									node := item.(*Node)
									mousePosition.resetMousePosition(x, y)
									if node.parent != nil && node.key == "" && !node.connected {
										showConnectAction.SetVisible(true)
										showReconnectAction.SetVisible(false)
										showDisconnectAction.SetVisible(false)
										showSearchAction.SetVisible(false)
										showDeleteAction.SetVisible(true)
									} else if node.parent != nil && node.key == "" && node.connected {
										showConnectAction.SetVisible(false)
										showReconnectAction.SetVisible(true)
										showDisconnectAction.SetVisible(true)
										showSearchAction.SetVisible(true)
										showDeleteAction.SetVisible(true)
									} else {
										showConnectAction.SetVisible(false)
										showReconnectAction.SetVisible(false)
										showDisconnectAction.SetVisible(false)
										showSearchAction.SetVisible(false)
										showDeleteAction.SetVisible(false)
									}
								}
							}
						},
					},
					Composite{
						Layout:        VBox{MarginsZero: true},
						StretchFactor: 3,
						Children: []Widget{
							TextLabel{
								Text: "Key:",
							},
							TextEdit{
								Text:          "",
								CompactHeight: true,
								ReadOnly:      true,
								AssignTo:      &key,
							},
							TextLabel{
								Text: "Value:",
							},
							TextEdit{
								AssignTo: &value,
							},
						},
					},
				},
				RowSpan: 10,
			},
			Label{
				Text:          "Version:1.0",
				TextAlignment: AlignFar,
			},
		},
	}.Create()); err != nil {
		zap.L().Error("Error", zap.Error(err))
		panic(err)
	}

	mainWindow.Run()
}

func connect(node *Node) {
	if ec := etcdConfigs[node.name]; ec != nil {
		createWaitDialog()
		waitDlg.Show()
		go func() {
			defer func() {
				time.Sleep(200 * time.Millisecond)
				waitDlg.Accept()
			}()
			client, err := clientv3.New(clientv3.Config{
				Endpoints:   []string{ec.Endpoint},
				Username:    ec.Username,
				Password:    ec.Password,
				DialTimeout: 2 * time.Second,
			})
			if err != nil {
				zap.L().Error("Error", zap.Error(err))
				walk.MsgBox(mainWindow, "Error", "Connect failed.", walk.MsgBoxIconError)
				return
			}
			timeoutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, err = client.Status(timeoutCtx, ec.Endpoint)
			if err != nil {
				zap.L().Error("Error", zap.Error(err))
				walk.MsgBox(mainWindow, "Error", "Connect failed.", walk.MsgBoxIconError)
				return
			}
			ec.Client = client
			resp, err := client.Get(context.Background(), "/",
				clientv3.WithPrefix(), clientv3.WithKeysOnly())
			if err != nil {
				zap.L().Error("Error", zap.Error(err))
				walk.MsgBox(mainWindow, "Error", "Connect failed.", walk.MsgBoxIconError)
				return
			}
			if resp != nil && resp.Kvs != nil {
				for _, v := range resp.Kvs {
					keys := strings.Split(strings.TrimPrefix(string(v.Key), "/"), "/")
					node.addNode(keys, string(v.Key), node.name)
				}
			}
			node.refreshNodeIcon("img/connected.ico")
			treeModel.PublishItemsReset(node)
			if node.children != nil {
				if err := nodes.SetExpanded(node, true); err != nil {
					zap.L().Error("Error", zap.Error(err))
					walk.MsgBox(mainWindow, "Error", "Connect failed.", walk.MsgBoxIconError)
					return
				}
			}
			key.SetText("")
			value.SetText("")
			node.connected = true
		}()
	} else {
		// TODO Error
	}
}

func search(searchKey string, node *Node) {
	if ec := etcdConfigs[node.name]; ec != nil && ec.Client != nil {
		node.children = nil
		resp, err := ec.Client.Get(context.Background(), searchKey,
			clientv3.WithPrefix(), clientv3.WithKeysOnly())
		if err != nil {
			zap.L().Error("Error", zap.Error(err))
			walk.MsgBox(mainWindow, "Error", "Search failed.", walk.MsgBoxIconError)
			return
		}
		if resp != nil && resp.Kvs != nil {
			for _, v := range resp.Kvs {
				keys := strings.Split(strings.TrimPrefix(string(v.Key), "/"), "/")
				node.addNode(keys, string(v.Key), node.name)
			}
		}
		if searchKey == "" {
			node.refreshNodeIcon("img/connected.ico")
		} else {
			node.refreshNodeIcon("img/search.ico")
		}
		treeModel.PublishItemsReset(node)
		if node.children != nil {
			if err := nodes.SetExpanded(node, true); err != nil {
				zap.L().Error("Error", zap.Error(err))
				walk.MsgBox(mainWindow, "Error", "Search failed.", walk.MsgBoxIconError)
				return
			}
		}
		key.SetText("")
		value.SetText("")
	} else {
		// TODO
	}
}

func disconnect(node *Node) {
	node.children = nil
	if ec := etcdConfigs[node.rootName]; ec != nil && ec.Client != nil {
		ec.Client.Close()
		ec.Client = nil
	}
	node.refreshNodeIcon("img/unconnected.ico")
	treeModel.PublishItemsReset(node)
	node.connected = false
}
