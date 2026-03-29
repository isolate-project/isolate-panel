const fs = require('fs');
const path = require('path');

const directory = './src/pages';
const files = fs.readdirSync(directory).filter(file => file.endsWith('.tsx'));

for (const file of files) {
  const filePath = path.join(directory, file);
  let content = fs.readFileSync(filePath, 'utf8');
  
  // 1. Update imports
  if (content.includes("from '../components/ui/Card'") || content.includes("from '../../components/ui/Card'")) {
    content = content.replace(
      /import\s+{\s*Card\s*}\s+from\s+['"]\.\.\/components\/ui\/Card['"]/g,
      "import { Card, CardContent, CardHeader, CardTitle, CardDescription, CardFooter } from '../components/ui/Card'"
    );
    content = content.replace(
      /import\s+{\s*Card\s*}\s+from\s+['"]\.\.\/\.\.\/components\/ui\/Card['"]/g,
      "import { Card, CardContent, CardHeader, CardTitle, CardDescription, CardFooter } from '../../components/ui/Card'"
    );
  }

  // 2. Replace simple <Card>...</Card>
  // This is tricky because we need to replace the opening tag and closing tag, 
  // but wrap the children in <CardContent>
  // Let's use a simpler regex for basic `<Card>` -> `<Card><CardContent>`
  // This might not perfectly catch props like `padding="none"` but we'll try our best.
  
  content = content.replace(/<Card(\s*[^>]*)>/g, '<Card$1>\n      <CardContent className="p-6">');
  content = content.replace(/<\/Card>/g, '      </CardContent>\n    </Card>');

  // 3. Fix the padding="none" issue
  content = content.replace(/<CardContent\s+className="p-6">(\s*)((?:.|\n)*?)padding="none"((?:.|\n)*?)<\/CardContent>/gi, 
                            '<CardContent className="p-0">$1$2$3</CardContent>');
  content = content.replace(/padding="none"/g, '');
  content = content.replace(/padding="sm"/g, '');
  content = content.replace(/padding="md"/g, '');
  content = content.replace(/padding="lg"/g, '');

  fs.writeFileSync(filePath, content, 'utf8');
  console.log(`Updated ${file}`);
}
