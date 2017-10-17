# 出前館待ち時間確認アプリ for Google Home

ローカル

```
go run server.go
ngrok http 9090

```

AWS Lambda

```sh
cd lambda

project.jsonを作成＆編集

apex deploy

API Gatewayを設定

```

## Dialogflowの設定

* Entitiesにdialogflow/entitiesのjsonをそれぞれ設定
* Intentsにdialogflow/intentsのjsonをそれぞれ設定
* DialogflowのFulfillmentのWebhookにngrokやAPI Gatewayで作成したURLを設定
* IntegrationでGoogle Assistantを有効に。Welcome Intentにmain.jsonのintent、残り２つをAdditional triggering intentsに設定。Update Draft実行。

このへんのimport作業は動作確認してません。

## 動作確認

* Google Homeに「OK Google, テスト用アプリに繋いで」

* アプリが起動するので会話する

- もつ鍋は何分？ -> もつ鍋は90分待ちです。
- インドカレーは？ -> インドカレーは58分待ちです。
- １番早いのは？ -> １番早いのは中華で35分です。
- 終了 -> 終了します（会話終了）


店のリストはgoのソースに書かれてるので好きな店に書き換えてください。
Actions on GoogleのSimulatorを使うとGoogle Homeがなくても動作します。