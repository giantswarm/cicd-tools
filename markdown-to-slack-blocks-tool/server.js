const mack = require('@tryfabric/mack');

async function run() {
  let input = process.env.INPUT
      .replace("\\n", "\n")
      .replace(/<@([A-Za-z0-9]+)>/g, "&lt;@$1&gt;") // User mentions
      .replace(/<!([A-Za-z0-9\|\^]+)>/g, "&lt;!$1&gt;") // Group / Special mentions
      .replace(/<#([A-Za-z0-9]+)>/g, "&lt;#$1&gt;") // Channels
    
  let blocks = await mack.markdownToBlocks(input);
  blocks = blocks.map(b => {
    if (b.text && b.text.text) {
      b.text.text = b.text.text
          .replace(/&lt;@([A-Za-z0-9]+)&gt;/g, "<@$1>") // User mentions
          .replace(/&lt;!([A-Za-z0-9\|\^]+)&gt;/g, "<!$1>") // Group / Special mentions
          .replace(/&lt;#([A-Za-z0-9]+)&gt;/g, "<#$1>") // Channels
    }
    return b;
  })
  console.log(JSON.stringify(blocks)); 
}

run();
