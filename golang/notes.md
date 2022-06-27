注意事项
------------



1、`return` 语句并不意味着 `RET` 指令，而仅仅是更新返回值而已

```go
func apple(v int) (int, error) {
    defer a()
    defer b()
    
    return 1, nil // MOVQ r1 = 1
    			  // MOVQ r2 = nil
    
    			  // CALL defer
    			  // RET
}
```

