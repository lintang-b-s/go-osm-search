package searcher

import "fmt"

type Deque struct {
	items []int
}

func NewDeque(items []int) Deque {
	return Deque{items}
}

func (d *Deque) GetSize() int {
	return len(d.items)
}

func (d *Deque) PushFront(item int) {
	d.items = append([]int{item}, d.items...)
}

func (d *Deque) PushBack(item int) {
	d.items = append(d.items, item)
}

func (d *Deque) PopFront() (int, bool) {
	if len(d.items) == 0 {
		return 0, false
	}
	frontElement := d.items[0]
	d.items = d.items[1:]
	return frontElement, true
}

func (d *Deque) PopBack() (int, bool) {
	if len(d.items) == 0 {
		return 0, false
	}
	rearElement := d.items[len(d.items)-1]
	d.items = d.items[:len(d.items)-1]
	return rearElement, true
}

func shuntingYardRPN(tokens []int) []int {
	precedence := make(map[int]int)
	precedence[-1] = 2 // AND
	precedence[-2] = 0 // (
	precedence[-3] = 0 // )
	precedence[-4] = 1 // OR
	precedence[-5] = 3 // NOT

	output := make([]int, 0, len(tokens))
	stack := []int{}

	for _, token := range tokens {
		if token == -2 {
			stack = append(stack, -2)
		} else if token == -3 {
			// pop
			n := len(stack) - 1
			operator := stack[n]
			stack = stack[:n]

			for operator != -2 {
				output = append(output, operator)
				// pop
				n = len(stack) - 1
				operator = stack[n]
				stack = stack[:n]
			}
		} else if _, ok := precedence[token]; ok {
			if len(stack) != 0 {
				n := len(stack) - 1
				operator := stack[n]

				for len(stack) != 0 && precedence[token] < precedence[operator] {
					output = append(output, operator)

					n = len(stack) - 1
					stack = stack[:n]
					if len(stack) != 0 {
						n = len(stack) - 1
						operator = stack[n]
					}
				}
			}

			stack = append(stack, token)
		} else {
			// term
			output = append(output, token)
		}
	}

	for len(stack) != 0 {
		n := len(stack) - 1
		token := stack[n]
		stack = stack[:n]
		output = append(output, token)
	}
	return output
}

// processQuery. process query -> return hasil boolean query (AND/OR/NOT) berupa posting lists (docIDs)
func (se *Searcher) processQuery(rpnDeque Deque) ([]int, error) {
	operator := map[int]struct{}{
		-1: struct{}{},
		-5: struct{}{},
		-4: struct{}{},
	}
	postingListStack := [][]int{}
	for rpnDeque.GetSize() != 0 {
		token, valid := rpnDeque.PopFront()
		if !valid {
			return []int{}, fmt.Errorf("rpn deque size is 0")
		}

		if _, ok := operator[token]; !ok {
			postingList, err := se.MainIndexNameField.GetPostingList(token)
			if err != nil {
				return []int{}, fmt.Errorf("error when get posting list skip list: %w", err)
			}
			postingListStack = append(postingListStack, postingList)
		} else {

			if token == -1 {
				// AND
				right := postingListStack[len(postingListStack)-1]
				postingListStack = postingListStack[:len(postingListStack)-1]
				left := postingListStack[len(postingListStack)-1]
				postingListStack = postingListStack[:len(postingListStack)-1]

				postingListIntersection := PostingListIntersection2(left, right)

				postingListStack = append(postingListStack, postingListIntersection)
			} else if token == -4 {
				// OR
				// NOT IMPLEMENTED YET
			} else {
				// NOT
				// NOT IMPLEMENTED YET
			}
		}
	}

	docIDsResult := postingListStack[len(postingListStack)-1]

	return docIDsResult, nil
}

func PostingListIntersection2(a, b []int) []int {

	idx1, idx2 := 0, 0
	result := []int{}

	for idx1 < len(a) && idx2 < len(b) {
		if a[idx1] < b[idx2] {
			idx1++
		} else if b[idx2] < a[idx1] {
			idx2++
		} else {
			result = append(result, a[idx1])
			idx1++
			idx2++
		}
	}
	return result
}
