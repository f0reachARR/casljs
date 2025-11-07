# CASL2 アセンブラ/ COMET2 エミュレータ for KIT 言語処理プログラミング

CASL2, COMET2 の JavaScript 実装です．
オリジナルの perl 版は https://github.com/omzn/casl2

## ブラウザ版

### index.html

* 開く Web ページ

### casl2comet2.js

* 状況: stable
  * jqueryterminal.js を利用して，Web ページ上の簡易ターミナルで動かすようにした． 
    CLIインタラクティブインターフェイスがそのまま利用できる．
  * CASL2のテキストエリアにファイルをドラッグアンドドロップできるようになった．

## コマンドライン版

### c2c2 (Go版) - 推奨

* 状況: stable
* Go言語で実装された高速なコマンドライン版です

#### ビルド方法

```bash
go build -o c2c2 .
```

#### 使用方法

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
```  

```bash
# 例：プログラムをアセンブル＆実行
./c2c2 -n -r -q caslfile.cas

# 例：事前に入力値を指定して実行
./c2c2 -n -Q sample.cas 10 20 30
```

#### テスト

```bash
# Go テストの実行
go test -v

# すべてのサンプルをテスト (28個のテストケース)
go test -v -run TestC2C2Samples
```

### c2c2.js (Node.js版)

* 状況: stable
```
Usage: c2c2 [options] <casl2file> [input1 ...]

Options:
  -V, --version   output the version number
  -a, --all       [casl2] show detailed info
  -c, --casl      [casl2] apply casl2 only
  -r, --run       [comet2] run immediately
  -n, --nocolor   [casl2/comet2] disable color messages
  -q, --quiet     [casl2/comet2] be quiet
  -Q, --QuietRun  [comet2] be QUIET! (implies -q and -r)
  -h, --help      display help for command
```  

* ブラウザ版と基本的な動作は同じなので，動作確認にはこちらを使った方が早い．

```
$ node c2c2 -n -r -q caslfile.cas
```
とすると，入出力のプロンプト以外の出力は抑制され(-q)，文字色は取り除かれ(-n)，プログラムをアセンブル後，すぐに実行が始まる(-r)．
また，動作確認などのため，あらかじめ`IN> `に渡すべき入力をファイル名以降の`[input ...]`に記述できる．

## 独自拡張(CASL2)

* ラベルにはスコープがあります．スコープはプログラム内(START 命令から END 命令で囲まれた部分)のみです．
* CALL 命令にもスコープが効きますが，CALL だけは別プログラムの開始ラベル(START 命令のラベル)まで参照できます．
* 簡単のため，MULA (算術乗算), MULL (論理乗算), DIVA (算術除算), DIVL (論理除算)を実装しています．利用方法は ADDA, ADDL 等とほぼ同じです．
* DC 命令で文字列を確保すると，最後に0(ヌル文字)が1文字追加されます．(文字列の終わりを容易に判定するため)
* ラベルは「英大文字，英小文字，$, _, %, . 」のいずれかで始まり，「英大文字，英小文字，数字，$, _, %, . 」を含む長さ制限の無い文字列で表します．
* ラベルのみの行を許容します．

## 独自拡張(COMET2)

* DIVA, DIVL については，0 除算を行おうとすると ZF と OF が同時に立って，メッセージを表示した後，プログラムは続行します．プログラム側でフラグを通じて0除算のチェックが必要です．

## 利用ライブラリ

* jQueryTerminal: https://terminal.jcubic.pl/
* encode.js: https://github.com/polygonplanet/encoding.js
* bcralnit.js: https://github.com/bachors/bcralnit.js
* pure.css

## License

* jqueryterminal

The MIT License (MIT)

Copyright (c) 2014 Erik Österberg

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

* encode.js

MIT License

Copyright (c) 2012 polygonplanet

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

* bcralnit.js

MIT License

Copyright (c) 2017 Ican Bachors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

* casl2/comet2

CASL assembler / COMET emulator
Copyright (C) 1998-2000 Hiroyuki Ohsaki (oosaki (at) ist.osaka-u.ac.jp)

CASL II assembler / COMET II emulator
Copyright (C) 2001-2023 Osamu Mizuno (o-mizuno (at) kit.ac.jp)

This program is free software; you can redistribute it and/or modify it
under the terms of the GNU General Public License as published by the
Free Software Foundation; either version 2 of the License, or (at your
option) any later version.

This program is distributed in the hope that it will be useful, but
WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General
Public License for more details.

You should have received a copy of the GNU General Public License along
with this program; if not, write to the Free Software Foundation, Inc.,
675 Mass Ave, Cambridge, MA 02139, USA.

* pure.css

Software License Agreement (BSD License)
========================================

Copyright 2013 Yahoo! Inc.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

    * Redistributions of source code must retain the above copyright
      notice, this list of conditions and the following disclaimer.

    * Redistributions in binary form must reproduce the above copyright
      notice, this list of conditions and the following disclaimer in the
      documentation and/or other materials provided with the distribution.

    * Neither the name of the Yahoo! Inc. nor the
      names of its contributors may be used to endorse or promote products
      derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL YAHOO! INC. BE LIABLE FOR ANY
DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
