package main

import "fmt"

func main() {
	fmt.Println("examples hello!")
}

func reverseList(head *ListNode) *ListNode {
	newHead := &ListNode{}

	for p := head; p != nil; {
		np := p                //将当前元素保存
		p = p.Next             //p 指针后移
		np.Next = newHead.Next // 将当前元素指针指向新链表的第一个元素
		newHead.Next = np      // head 指向当前元素
	}

	return newHead.Next
}
