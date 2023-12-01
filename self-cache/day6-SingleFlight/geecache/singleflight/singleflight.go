package singleflight

import "sync"
type call struct{
	wg sync
}