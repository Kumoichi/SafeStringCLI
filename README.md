# SafeStringCLI

GoのAST（抽象構文木）を使用してfmt.Errorへのstring型引数を自動的にerrorsx.SafeStringでラップするCLIツールです。

## 概要

このツールは、Go言語の静的解析を行い、`fmt.Errorf`の`%v`および`%s`フォーマット指定子に渡されるstring型変数を自動的に`errorsx.SafeString`でラップして、より安全なエラーメッセージを生成します。

## ファイル構成

- `safestring_fixer.go` - メインのCLIツール（AST解析・変換処理）
- `errorsx/errorsx.go` - SafeString型の定義（ダミー実装）
- `test_target.go` - 変換テスト用のサンプルファイル

## 使用方法

```bash
# ツールを実行
go run safestring_fixer.go
```

実行すると、現在のディレクトリ内のすべての`.go`ファイル（`_test.go`と`_fix.go`を除く）を解析し、変換が必要な箇所を`errorsx.SafeString`でラップして、`_fix.go`という接尾辞を付けた新しいファイルを生成します。

## 変換例

### 変換前（test_target.go）
```go
package main

import "fmt"

func main() {
    s := "hoge"
    fmt.Println(fmt.Errorf("%v", s))
    fmt.Println(fmt.Errorf("%s", s))
}
```

### 変換後（test_target_fix.go）
```go
package main

import (
    "fmt"
    "github.com/Kumoichi/SafeStringCLI/errorsx"
)

func main() {
    s := "hoge"
    fmt.Println(fmt.Errorf("%v", errorsx.SafeString(s)))
    fmt.Println(fmt.Errorf("%s", errorsx.SafeString(s)))
}
```

## 機能

- ✅ `fmt.Errorf`の`%v`および`%s`フォーマット指定子を対象とする
- ✅ string型変数のみを変換（他の型はスキップ）
- ✅ 既に`errorsx.SafeString`でラップされている場合はスキップ
- ✅ 必要に応じて`errorsx`パッケージのimportを自動追加
- ✅ ディレクトリ内のすべてのGoファイルを処理
- ✅ 元ファイル名に`_fix.go`接尾辞を付けて出力

## 技術的な詳細

このツールは以下のGo標準ライブラリを使用しています：

- `go/ast` - 抽象構文木の解析と操作
- `go/parser` - Goソースコードの解析
- `go/token` - トークンとポジション情報の管理
- `go/format` - フォーマットされたGoコードの出力