# CASL2 アセンブラ/ COMET2 エミュレータ for KIT 言語処理プログラミング

CASL2, COMET2 の Go 実装です．  
オリジナルの perl 版は https://github.com/omzn/casl2

## コマンドライン版 (Go)

Go言語で実装された高速なコマンドライン版です。

### ビルド方法

```bash
go build -o c2c2 .
```

### 使用方法

```
Usage: c2c2 [options] <casl2file> [input1 ...]

Options:
  -V          output the version number
  -a          [casl2] show detailed info
  -c          [casl2] apply casl2 only
  -r          [comet2] run immediately
  -n          [casl2/comet2] disable color messages
  -q          [casl2/comet2] be quiet
  -Q          [comet2] be QUIET! (implies -q and -r)
  -dap port   [dap] start Debug Adapter Protocol server on specified TCP port
```  

```bash
# 例：プログラムをアセンブル＆実行
./c2c2 -n -r -q caslfile.cas

# 例：事前に入力値を指定して実行
./c2c2 -n -Q sample.cas 10 20 30

# 例：Debug Adapter Protocolサーバーを起動（ポート4711で待機）
./c2c2 -dap 4711
```

### Debug Adapter Protocol (DAP) サポート

c2c2は、エディタやIDEからのデバッグを可能にするDebug Adapter Protocolをサポートしています。

#### 使用方法

1. DAPサーバーを起動:
```bash
./c2c2 -dap 4711
```

2. エディタ/IDEからTCPポート4711経由でDAPプロトコルを使用してデバッグセッションを開始

#### VS Code拡張機能

VS Codeユーザー向けに、専用の拡張機能を提供しています：

```bash
cd vscode-casl2-debug
pnpm install
pnpm run compile
pnpm run package
```

詳細は [vscode-casl2-debug/README.md](vscode-casl2-debug/README.md) を参照してください。

#### サポートされる機能

- プログラムの起動 (launch)
- ステップ実行 (step, stepIn, stepOut)
- 継続実行 (continue)
- ブレークポイントの設定
- レジスタの検査 (PC, FR, GR0-GR7, SP)
- スタックトレースの表示
- 一時停止 (pause)

#### 注意事項

- 標準入出力は IN/OUT 命令のために使用されます
- DAPモード時は通常のインタラクティブモードは使用できません

### テスト

```bash
# Go テストの実行
go test -v

# すべてのサンプルをテスト (28個のテストケース)
go test -v -run TestC2C2Samples

# カバレッジ付きでテスト
go test -v -race -coverprofile=coverage.txt -covermode=atomic
```

## 特徴

- **高速**: コンパイルされた Go バイナリで動作
- **クロスプラットフォーム**: Windows, macOS, Linux で動作
- **依存関係なし**: 単一のバイナリファイルで動作
- **型安全**: Go の型システムによる安全性
- **テスト**: 28個のテストケースで検証済み

## 独自拡張(CASL2)

* ラベルにはスコープがあります．スコープはプログラム内(START 命令から END 命令で囲まれた部分)のみです．
* CALL 命令にもスコープが効きますが，CALL だけは別プログラムの開始ラベル(START 命令のラベル)まで参照できます．
* 簡単のため，MULA (算術乗算), MULL (論理乗算), DIVA (算術除算), DIVL (論理除算)を実装しています．利用方法は ADDA, ADDL 等とほぼ同じです．
* DC 命令で文字列を確保すると，最後に0(ヌル文字)が1文字追加されます．(文字列の終わりを容易に判定するため)
* ラベルは「英大文字，英小文字，$, _, %, . 」のいずれかで始まり，「英大文字，英小文字，数字，$, _, %, . 」を含む長さ制限の無い文字列で表します．
* ラベルのみの行を許容します．

## 独自拡張(COMET2)

* DIVA, DIVL については，0 除算を行おうとすると ZF と OF が同時に立って，メッセージを表示した後，プログラムは続行します．プログラム側でフラグを通じて0除算のチェックが必要です．

## 実装について

詳細な実装情報は [GO_README.md](GO_README.md) を参照してください。

## ライセンス

GPL v3 - 詳細は [COPYING](COPYING) ファイルを参照してください。
