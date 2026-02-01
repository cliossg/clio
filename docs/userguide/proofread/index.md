# Proofread

Proofread uses AI to review your writing and suggest corrections. It checks grammar, style, repetitions, and overused phrases, all from the content editor.

## Why use Proofread?

- **Catch errors**: Fix spelling, grammar, and punctuation mistakes
- **Improve clarity**: Get suggestions to tighten verbose sentences
- **Spot repetitions**: Find words or phrases repeated too close together
- **Avoid clichés**: Flag worn-out expressions that weaken your writing

## Quick Start

1. Open any content in the editor

2. Click the **Proofread** button in the toolbar

3. Review the corrections and click **Apply** to accept them

That's it. Your text is updated with the corrections.

## How It Works

When you click Proofread, Clio sends your text to an AI model that performs four integrated passes:

| Pass        | What it checks                               |
| ----------- | -------------------------------------------- |
| **Grammar** | Spelling, grammar, punctuation, syntax       |
| **Style**   | Clarity, flow, verbosity                     |
| **Echoes**  | Words or phrases repeated in close proximity |
| **Overuse** | Clichés and worn expressions                 |

The AI is conservative. It preserves your voice and makes the smallest change that fixes each problem.

### The Result View

After proofreading, you'll see:

- **Summary**: A brief description of what was changed
- **Corrected text**: The full text with corrections applied
- **Actions**:
  - **Apply**: Replace your text with the corrected version
  - **Changes**: View the detailed list of corrections
  - **Discard**: Keep your original text

### Viewing Corrections

Click **Changes** to see every correction grouped by type:

- **Grammar**: Spelling, punctuation, syntax fixes
- **Style**: Clarity and verbosity improvements
- **Echo**: Repeated words flagged
- **Overuse**: Clichés identified

Each correction shows the original text, the replacement, and an explanation of why it was changed.

## Proofreading a Selection

You can proofread just part of your content:

1. Select text in the editor
2. Click **Proofread**
3. Only the selected text is analyzed
4. **Apply** replaces just the selection

This is useful for revising a specific paragraph without reprocessing the entire article.

## Common Workflows

### Quick grammar check before publishing

1. Finish writing your content
2. Click **Proofread**
3. Review the summary
4. Click **Apply** if the corrections look good
5. Publish

### Careful review with explanations

1. Click **Proofread**
2. Click **Changes** to see the full list
3. Read through each correction and its explanation
4. Decide whether to **Apply** or **Discard**
5. If you discard, manually apply the corrections you agree with

### Iterative polishing

1. Proofread your content
2. Apply corrections
3. Read through the result
4. Proofread again if needed (diminishing returns expected)

## Multilingual Support

Proofread detects the language of your text automatically. It works with multiple languages and won't try to "correct" text into English if you're writing in another language.

The AI also respects quoted text and citations in other languages, leaving them unchanged.

## Configuration

### Setting up the API key

Proofread requires an OpenAI API key. You can configure it in two ways:

**Option 1: Environment variable**

Set `OPENAI_API_KEY` before starting Clio:

```bash
export OPENAI_API_KEY=sk-...
```

**Option 2: Configuration file**

Add to your `config.yaml`:

```yaml
llm:
  api_key: sk-...
```

### Changing the model

By default, Proofread uses `gpt-4o`. To use a different model:

```yaml
llm:
  model: gpt-4o-mini
```

Or via environment variable:

```bash
export CLIO_LLM_MODEL=gpt-4o-mini
```

## Troubleshooting

### "LLM API key not configured"

The API key isn't set. See [Configuration](#configuration) above.

### Proofread takes a long time

The AI model needs time to analyze your text, especially for longer content. For very long articles, consider proofreading section by section using text selection.

### Corrections seem wrong

The AI is generally accurate but not perfect. Always review corrections before applying. Use the **Changes** view to understand why each correction was suggested.

If a correction doesn't fit your style or intent, click **Discard** and keep your original text.

### No corrections found

If your text is already clean, Proofread will report "No corrections needed." This is normal for well-edited content.
