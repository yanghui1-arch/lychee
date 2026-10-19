import { expect, test } from 'bun:test'
import { readFileSync, readdirSync } from 'node:fs'
import { join, relative } from 'node:path'

const source = join(import.meta.dir, '../src')

function files(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name)
    return entry.isDirectory() ? files(path) : [path]
  })
}

test('component styles use Tailwind rather than separate stylesheets', () => {
  expect(
    files(source)
      .filter((file) => file.endsWith('.css'))
      .map((file) => relative(source, file).replaceAll('\\', '/')),
  ).toEqual(['styles/globals.css'])
  for (const file of files(source).filter((file) => /\.[jt]sx?$/.test(file))) {
    const imports = [
      ...readFileSync(file, 'utf8').matchAll(/import\s+['"]([^'"]+\.css)['"]/g),
    ].map((match) => match[1])
    expect(imports).toEqual(
      file.endsWith('main.tsx') ? ['./styles/globals.css'] : [],
    )
  }
})

test('the Tailwind entry contains only framework, theme and font configuration', () => {
  const entry = readFileSync(join(source, 'styles/globals.css'), 'utf8')
  const configuration = entry
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/@(?:import|custom-variant)\s+[^;]+;/g, '')
    .replace(/@font-face\s*\{[^}]*\}/g, '')
    .replace(
      /(?:@theme\s+inline|:root|\.dark)\s*\{(?:\s*--[\w-]+\s*:[^;]+;)*\s*\}/g,
      '',
    )
  expect(configuration.trim()).toBe('')
})
