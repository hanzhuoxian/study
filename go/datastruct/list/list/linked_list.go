package main

// Element 链表的节点
type Element struct {
	next  *Element
	list  *List
	Value interface{}
}

// List 单链表
type List struct {
	head Element
	tail *Element
	len  int
}

// New 创建一个新列表
func New() *List {
	return new(List).Init()
}

// Init 初始化列表或者清除列表 l
func (l *List) Init() *List {
	l.head.next = &l.head
	l.tail = &l.head
	l.len = 0
	return l
}
