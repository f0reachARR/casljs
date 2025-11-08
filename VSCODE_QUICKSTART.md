# VS Code Extension Quick Start

## インストール

### 方法1: VSIXファイルから（推奨）

1. 拡張機能をビルド:
```bash
cd vscode-casl2-debug
pnpm install
pnpm run compile
pnpm run package
```

2. VS Codeで拡張機能をインストール:
   - VS Codeを開く
   - 拡張機能ビュー (Ctrl+Shift+X)
   - "..." メニュー → "VSIXからのインストール..."
   - `casl2-debug-1.0.0.vsix` を選択

### 方法2: 開発モード

```bash
cd vscode-casl2-debug
pnpm install
pnpm run compile
```

VS Codeで `vscode-casl2-debug` フォルダを開き、F5を押して拡張機能開発ホストを起動。

## 使い方

### 1. デバッグセッションの開始

1. CASL2ファイル (`.cas`) を開く
2. `F5` を押す
3. エントリポイントで自動停止

### 2. ブレークポイントの設定

行番号の左側をクリックしてブレークポイントを設定（赤い点が表示されます）。

### 3. デバッグコントロール

- **F5**: 継続実行
- **F10**: ステップオーバー
- **F11**: ステップイン
- **Shift+F11**: ステップアウト
- **Shift+F5**: デバッグ停止

### 4. 変数の確認

左側の「変数」パネルで以下のレジスタを確認できます：
- PC (プログラムカウンタ)
- FR (フラグレジスタ)
- GR0-GR7 (汎用レジスタ)
- SP (スタックポインタ)

## カスタム設定

`.vscode/launch.json` を作成してデバッグ設定をカスタマイズ:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "type": "casl2",
      "request": "launch",
      "name": "Debug CASL2",
      "program": "${file}",
      "stopOnEntry": true,
      "debugServer": 4711,
      "c2c2Path": "c2c2"
    }
  ]
}
```

### 設定オプション

- `program`: デバッグするCASL2ファイルのパス（`${file}` で現在のファイル）
- `stopOnEntry`: エントリポイントで停止するか（デフォルト: true）
- `debugServer`: DAPサーバーのTCPポート（デフォルト: 4711）
- `c2c2Path`: c2c2実行ファイルのパス（デフォルト: "c2c2"）

## トラブルシューティング

### c2c2が見つからない

c2c2がPATHに含まれているか確認するか、フルパスを指定:
```json
{
  "c2c2Path": "/full/path/to/c2c2"
}
```

### ポートが使用中

別のポートを使用:
```json
{
  "debugServer": 4712
}
```

### 拡張機能が有効にならない

- VS Codeのバージョンを確認（1.75.0以上が必要）
- `out/` ディレクトリにコンパイル済みJavaScriptが存在するか確認
- 出力パネル（表示 → 出力 → 拡張機能ホスト）でエラーを確認

## サンプルプログラム

`examples/simple_add.cas` を使ってテスト:

```casl2
MAIN    START
        LD      GR0, =10
        LD      GR1, =20
        ADDA    GR0, GR1
        ST      GR0, RESULT
        RET
RESULT  DS      1
        END
```

## 詳細情報

- ユーザーガイド: `vscode-casl2-debug/README.md`
- 開発者ガイド: `vscode-casl2-debug/DEVELOPMENT.md`
- 変更履歴: `vscode-casl2-debug/CHANGELOG.md`
