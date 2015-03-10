package director

import (
	"container/list"
	"fmt"
	"strings"
	"testing"
)

var (
	testPrefixes = []string{
		"ce633eca",
		"d01e78583b94e00e3f1df",
		"bb603c25",
		"872a7d239c8de0344074a64",
		"1f8d9b2e8c8324c599b",
		"ba68b1c83e32003c1d",
		"60b42762a",
		"08f15236fd16784fb23991b62e",
		"113c69ede64cf63dba38",
		"7539c985a791fe621b3841c4169",
		"8c2371",
		"cd7829ca1e231b23e215317eb40644",
		"aafc5675380fe20e032cb8fe",
		"484015cd",
		"9edced1c8204a693e334d",
		"ccc88bf869198aa7b",
		"5651a650c6ad952d226057",
		"07",
		"351f0e",
		"530cbcfe115",
		"6136010ca0fd0a73aea2cb",
		"995a11d4cc92290f9b0a84",
		"c5e192",
		"be0c5643e2c7866480",
		"c3de5673663f",
		"1a2bb8560b313d31c88",
		"3db05a50f5a60ac3d3a6",
		"642d51d9005aac00782aaba7bb1",
		"f4fa6a",
		"8b2cc9a3147cc4d32ef05ffa8f",
		"7dd2259dd5768d542316410",
		"b979fbec6f046a1af9eec",
		"378828e7a5",
		"450c6672f8ee163f078",
		"76e504e2bc0eb3e48",
		"544f38b5e7d065",
		"50698281280e",
		"21e24493",
		"68ed53a2c600",
		"85caed951c9d",
		"c0b918c1d0599748e0d676",
		"127dcb3dfdd5886c69c79c",
		"ded89c5fcd58bd0bec414a72b855",
		"4db7b23ef282991a",
		"3cc3dfcf956fedc3c7d37429aa1ab4",
		"8c4caad37b",
		"c1",
		"0062",
		"4f22277a",
		"6f",
		"faab272194c99f46",
		"c92ead0e3050bb43169ff87c700fc41",
		"4142c156837ffdd923",
		"2e37c168368ee7c9526ed2b8a77590b",
		"b77f65a",
		"64727d2208b54bd7f82c",
		"af660167ffae9d",
		"edd8f161a93b4b5e1d",
		"c6266c1569c9cc29",
		"6af68527d70f5eb0",
		"82e3f4749b05bc60eec53840547",
		"3e05a1",
		"6191127d31699",
		"0fbe8c21593db558cfbb791ddc04972",
	}
)

func TestDomain(t *testing.T) {

	d := newDomain()

	for _, prefix := range testPrefixes {
		d.setPrefix(prefix, fmt.Sprintf("service-%s", prefix))
	}

	for i := 0; i < len(d.pathPrefixesList)-1; i++ {
		if len(d.pathPrefixesList[i]) < len(d.pathPrefixesList[i+1]) {
			t.Fatal("domain prefix list is out of order")
		}
	}

	for i := len(testPrefixes) - 1; i > 0; i-- {

		service, err := d.pick(testPrefixes[i])
		if err != nil {
			t.Fatal(err)
		}

		if service != fmt.Sprintf("service-%s", testPrefixes[i]) {
			t.Fatal("incorrect service match")
		}
	}
}

func BenchmarkSlice(b *testing.B) {

	for j := 0; j < b.N; j++ {

		pathPrefixes := make(map[string]string)
		pathPrefixesList := make([]string, 0)

		for _, prefix := range testPrefixes {
			if _, ok := pathPrefixes[prefix]; !ok {

				// Save a temporary reference to the list and create a new list.
				tmpPathPrefixesList := pathPrefixesList
				pathPrefixesList = make([]string, len(pathPrefixesList)+1)

				// Find the correct index for the prefix, copying all values up to that point.
				i := 0
				for ; i < len(tmpPathPrefixesList) && len(tmpPathPrefixesList[i]) > len(prefix); i++ {
					pathPrefixesList[i] = tmpPathPrefixesList[i]
				}

				// Set the prefix.
				pathPrefixesList[i] = prefix

				// Copy the remaining values from the old list.
				for ; i < len(tmpPathPrefixesList); i++ {
					pathPrefixesList[i+1] = tmpPathPrefixesList[i]
				}
			}

			pathPrefixes[prefix] = "service"
		}
	}
}

func BenchmarkSliceSeek(b *testing.B) {

	pathPrefixes := make(map[string]string)
	pathPrefixesList := make([]string, 0)

	for _, prefix := range testPrefixes {
		if _, ok := pathPrefixes[prefix]; !ok {

			// Save a temporary reference to the list and create a new list.
			tmpPathPrefixesList := pathPrefixesList
			pathPrefixesList = make([]string, len(pathPrefixesList)+1)

			// Find the correct index for the prefix, copying all values up to that point.
			i := 0
			for ; i < len(tmpPathPrefixesList) && len(tmpPathPrefixesList[i]) > len(prefix); i++ {
				pathPrefixesList[i] = tmpPathPrefixesList[i]
			}

			// Set the prefix.
			pathPrefixesList[i] = prefix

			// Copy the remaining values from the old list.
			for ; i < len(tmpPathPrefixesList); i++ {
				pathPrefixesList[i+1] = tmpPathPrefixesList[i]
			}
		}

		pathPrefixes[prefix] = "service"
	}

	b.ResetTimer()

	for j := 0; j < b.N; j++ {
		for _, prefix := range testPrefixes {
			for _, value := range pathPrefixesList {
				if strings.HasPrefix(value, prefix) {
					break
				}
			}
		}
	}
}

func BenchmarkList(b *testing.B) {

	for j := 0; j < b.N; j++ {

		pathPrefixes := make(map[string]string)
		pathPrefixesList := list.New()

		for _, prefix := range testPrefixes {
			if _, ok := pathPrefixes[prefix]; !ok {

				if pathPrefixesList.Len() == 0 {
					pathPrefixesList.PushFront(prefix)
				} else {

					e := pathPrefixesList.Front()
					for ; e.Next() != nil; e = e.Next() {
						if len(e.Value.(string)) <= len(prefix) {
							break
						}
					}

					pathPrefixesList.InsertBefore(prefix, e)
				}
			}

			pathPrefixes[prefix] = "service"
		}
	}
}

func BenchmarkListSeek(b *testing.B) {

	pathPrefixes := make(map[string]string)
	pathPrefixesList := list.New()

	for _, prefix := range testPrefixes {
		if _, ok := pathPrefixes[prefix]; !ok {

			if pathPrefixesList.Len() == 0 {
				pathPrefixesList.PushFront(prefix)
			} else {

				e := pathPrefixesList.Front()
				for ; e.Next() != nil; e = e.Next() {
					if len(e.Value.(string)) <= len(prefix) {
						break
					}
				}

				pathPrefixesList.InsertBefore(prefix, e)
			}
		}

		pathPrefixes[prefix] = "service"
	}

	b.ResetTimer()

	for j := 0; j < b.N; j++ {
		for _, prefix := range testPrefixes {
			for e := pathPrefixesList.Front(); e != nil; e = e.Next() {
				if strings.HasPrefix(e.Value.(string), prefix) {
					break
				}
			}
		}
	}
}
