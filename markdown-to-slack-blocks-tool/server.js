const mack = require('@tryfabric/mack');

async function run() {
  let blocks = await mack.markdownToBlocks(process.env.INPUT.replace("\\n", "\n"));
  blocks = blocks.map(b => {
    if (b.text && b.text.text) {
      b.text.text = b.text.text
          .replace(/&lt;@([A-Za-z0-9]+)&gt;/g, "<@$1>") // User mentions
          .replace(/&lt;!([A-Za-z0-9]+\^[A-Za-z0-9]+)&gt;/g, "<!$1>") // Group mentions
          .replace(/&lt;!([A-Za-z0-9\|]+)&gt;/g, "<!$1>") // Special mentions
          .replace(/&lt;#([A-Za-z0-9]+)&gt;/g, "<#$1>"); // Channels
    }
    return b;
  })
  console.log(JSON.stringify(blocks));
}

run();
