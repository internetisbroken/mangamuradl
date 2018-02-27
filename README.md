# mangamuradl

## ダウンロード

windows(x64)ビルドは以下のリリースページの mangamuradl.zip から取得可能

https://github.com/internetisbroken/mangamuradl/releases

## 使い方 (GUI)

mangamuragui.exe を実行し、
右上の入力ボックスにpageid（URLまたは数字）を入力した後、左上のStartボタンを押す。
結果が下のテキストボックスに表示される。

動作中にファイアウォールのメッセージが出る場合、キャンセル（不許可）で問題ない。

入力例:

```
hxxp://mangamura.org/?p=1234567890
1234567890
```

必要であればオプションを付ける。
詳細は"コマンドラインオプション" 参照。


## mangamuradl の使い方 (コマンドライン)

Powershellかコマンドプロンプト等で

```
.\mangamuradl [options...] pageid
```

例：
```
.\mangamuradl 1234567890
```

実行すると「img/タイトル名」のフォルダが作成され、その下に画像がDLされる。


### pageid

以下のような感じを入力。
- hxxp://mangamura.org/?p=1234567890
- 1234567890


### コマンドラインオプション

~~-cookie=xxxx : 画像URL要求時のcookieを指定する。値は"cookie(acookie4)の調べ方"参照。~~

- cookieオプションは廃止したので、指定する場合はmangamuradl.iniのacookie4を書き換える。


## getcookie の使い方

Chromeブラウザを使用し、認証を行うプログラム。

GUIまたはコマンドツールを実行した際、認証が必要な場合に実行される。

デバッグ用ウェブブラウザが立ち上がるので、そこで認証を完了させるとcookieがファイル(mangamuradl.ini)に保存される。

明示的に実行したい場合は、Powershellかコマンドプロンプト上で

```
.\getcookie
```

を実行する。

本ツールの実行には、Chrome ブラウザと ChromeDriver が必要。
ChromeDriver は以下のサイトからダウンロードし、getcookie.exeと同じディレクトリに、
chromedriver.exeを解凍して配置する。

ChromeDriver: https://sites.google.com/a/chromium.org/chromedriver/downloads

ファイアウォールのメッセージが出る場合、キャンセル（不許可）で問題ない。



### cookie(acookie4)の調べ方

getcookieを使用する限り、この作業は不要

- ウェブブラウザ(Chrome)にて、あらかじめ認証を行っておく
- mangamura.orgの任意のページを開く
- アドレスバーに以下を入力したのち表示された文字列がcookieで必要な値。
-- 単純にペーストすると"javascript:"が削除される場合があるので、削除された部分は手入力する

```
javascript:document.write($.cookie("acookie4"))
```

## 外部ツール

外部ツールでファイアウォールのメッセージが出る場合、キャンセル（不許可）で問題ない。

- convert (ImageMagic) pdf作成および分割ページの処理に必要
- phantomjs 分割ページの処理に必要
- chromedriver 認証ページの操作に必要

## トラブルシュート

- 動作がおかしい時はmangamuradl.iniとdbフォルダを削除してみて下さい
- リリースページに最新版がある場合は更新して下さい
