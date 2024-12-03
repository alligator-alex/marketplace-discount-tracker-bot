# Telegram bot app for tracking discounts and availability on RU marketplaces

Allows you to keep track of discounts on the desired products, as well as their restock.

### Marketplaces currently supported:
- [Ozon](https://www.ozon.ru/)
- [Wildberries](https://www.wildberries.ru/)

---

### Deployment

1. Create new Telegram bot using [@BotFather](https://t.me/BotFather).

2. Add list of commands to the bot:  
   ```
   trackproduct - keep track of discounts
   listproducts - show list of tracked products
   cancel - cancel current action
   help - show help
   ```

3. Copy `.env.example` file and rename it to `.env`.

4. Grab a token from *BotFather* and pase it as `TELEGRAM_BOT_TOKEN` variable in newly created `.env` file.

5. Run app in Docker
   - For the first lauch type `make prepare-and-up` command
   - For regular use just type `make up` command

6. Add your bot from step 1 to your Telegram contacts and then just use commands from step 2.

---

### Usage

To get started, you need to add a product to the bot.  
Enter `/trackproduct` command and then send the desired product URL to save it.

The bot will automatically check your saved URLs every 60 minutes in the background (interval could be changed in .env-file).  
If the price of any product has dropped or it's back in stock, the bot will send you a corresponding message.

To view the list of your tracked products, use the `/listproducts` command.  
If there are more than 5 results, pagination will be shown.  
While in list, you can also delete unwanted products by clicking a link like `/del_abCdEF1`.

To cancel any action, use `/cancel` command.  
**But you cannot cancel the background price/availability check**.
