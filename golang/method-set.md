Method Set
-----------------

cf. https://stackoverflow.com/questions/33587227/method-sets-pointer-vs-value-receiver

```plaintext
Values          Methods Receivers
-----------------------------------------------
T               (t T)
*T              (t T) and (t *T)

Methods Receivers    Values
-----------------------------------------------
(t T)                 T and *T
(t *T)                *T
```



### Golang FAQ

cf. https://go.dev/doc/faq#different_method_sets

> ### Why do T and *T have different method sets?
>
> As the [Go specification](https://go.dev/ref/spec#Types) says, the method set of a type `T` consists of all methods with receiver type `T`, while that of the corresponding pointer type `*T` consists of all methods with receiver `*T` or `T`. That means the method set of `*T` includes that of `T`, but not the reverse.
>
> This distinction arises because if an interface value contains a pointer `*T`, a method call can obtain a value by dereferencing the pointer, but if an interface value contains a value `T`, there is no safe way for a method call to obtain a pointer. (Doing so would allow a method to modify the contents of the value inside the interface, which is not permitted by the language specification.)
>
> Even in cases where the compiler could take the address of a value to pass to the method, if the method modifies the value the changes will be lost in the caller. As an example, if the `Write` method of [`bytes.Buffer`](https://go.dev/pkg/bytes/#Buffer) used a value receiver rather than a pointer, this code:
>
> ```
> var buf bytes.Buffer
> io.Copy(buf, os.Stdin)
> ```
>
> would copy standard input into a *copy* of `buf`, not into `buf` itself. This is almost never the desired behavior.