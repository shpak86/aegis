import puppeteer from 'puppeteer'

(async () => {
    const browser = await puppeteer.launch({ headless: false })
    const page = await browser.newPage()
    // Replace history.back action
    await page.evaluateOnNewDocument(() => {
        const originalGo = window.history.go;
        window.history.back = function () {
            document.location.href = "/"
            return;
        };
        window.history.go = function (delta) {
            if (delta < 0) {
                console.log('history.go() with negative values is blocked');
                return;
            }
            return originalGo.call(this, delta);
        };
    });
    // Go!
    await page.goto("http://127.0.0.1/",
        {
            waitUntil: "networkidle0"
        }
    )
    // Wait for button on the main page and while all the data is loaded
    await page.waitForSelector(".card-like-btn")
    await page.waitForNetworkIdle()
    // Show Antibot cookie
    const cookies = await browser.cookies()
    const AEGIS_TOKEN = cookies.filter(it => it.name === "AEGIS_TOKEN")
    console.log("AEGIS_TOKEN: " + AEGIS_TOKEN[0].value)
    // Like first article
    for (let i = 0; i < 100; i++) {
        const button = await page.$(".card-like-btn")
        await button.click()
        // await page.waitForNetworkIdle()
    }
})()
