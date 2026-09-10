# ga4 Provider

Google Analytics 4 の Admin API でプロパティ・データストリーム・Measurement Protocol API シークレット・
カスタムディメンションを管理する。

## 引数

- `credentials`（省略可、秘匿）: サービスアカウントの JSON キー。ファイルパスか JSON の中身そのもの。
  省略すると Application Default Credentials を使う。

資格情報の主体（サービスアカウントのメールアドレス、またはユーザー）を、GA4 のアカウントかプロパティに
「編集者」として追加しておくこと。
