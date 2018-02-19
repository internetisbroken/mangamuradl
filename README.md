# mangamuradl

## ダウンロード

以下のリンクから可能。

https://github.com/internetisbroken/mangamuradl/archive/master.zip

## 使い方 (GUI)

mangamuragui.exe を実行し、
右上の入力ボックスに漫画ID（URLまたは数字）とオプション(-cookie=xxxx)を入力した後、
左上のStartボタンを押す。
結果が下のテキストボックスに表示される。

入力例:
```
hxxp://mangamura.org/?p=1234567890 -cookie=xxxx
```

なおxxxxの部分は、mangamura.orgでのacookie4の値を入力する。

### cookie(acookie4)の調べ方

- ウェブブラウザにて、あらかじめ認証を行っておく
- mangamura.orgの任意のページを開く
- アドレスバーに以下を入力したのち表示された文字列がcookieで必要な値。
-- 単純にペーストすると"javascript:"が削除される場合があるので、削除された部分は手入力する

```
javascript:document.write($.cookie("acookie4"))
```

## 使い方 (コマンドライン)

Powershellかコマンドプロンプト開いて
```
.\mangamuradl.exe <ダウンロードしたい漫画ID> -cookie=xxxx
```

例：
```
.\mangamuradl.exe 1234567890 -cookie=xxxx
```

### コマンドラインオプション

-cookie=xxxx : 画像URL要求時のcookieを指定する。"cookieの調べ方"参照。


実行すると現在のディレクトリにタイトル名のフォルダが作成フォルダが作成され、その下に画像がDLされます。

## 漫画ID

以下のような感じを入力。
- hxxp://mangamura.org/?p=1234567890
- 1234567890

