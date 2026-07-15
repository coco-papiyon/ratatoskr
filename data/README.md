# Ratatoskr Test Data

ビューアの動作確認用データです。

## 確認項目

- Markdown の見出し、リスト、コードブロック
- CSV のヘッダー切り替え
- JSON / XML の構造化表示
- ソースコードのシンタックスハイライト
- テキスト表示
- 画像表示
- ZIP / TAR.GZ 内のフォルダ移動とファイル表示
- 圧縮ファイル内の SVG 画像表示

## 圧縮ファイル

- `compress/sample-archive.zip`
- `compress/sample-archive.tar.gz`

どちらにも `README.md`、`docs/notes.txt`、構造化データ、`images/test-image.svg` が含まれています。

```ts
const source = "local";
console.log(`preview: ${source}`);
```

## Markdown 記法サンプル

これは **太字**、*斜体*、~~取り消し線~~、[リンク](https://example.com) の表示確認です。

文章の中に `インラインコード` を置くこともできます。

![Ratatoskr image preview](images/test-image.svg)

### 表

| 種類 | 拡張子 | 表示方法 |
| --- | --- | --- |
| Markdown | `.md` | レンダリング |
| CSV | `.csv` | 表形式 |
| JSON | `.json` | 整形表示 |
| 画像 | `.svg` | 画像ビューア |

### 順序付きリスト

1. Local タブを選択する
2. フォルダを開く
3. ファイルを一覧から選択する

### 引用

> ファイルを開くまでの手順を減らし、内容をすぐ確認できるようにします。
>
> これは複数行の引用です。

### コードブロック

```json
{
  "source": "local",
  "preview": true,
  "encoding": "utf-8"
}
```

```bash
echo "Ratatoskr viewer"
```
