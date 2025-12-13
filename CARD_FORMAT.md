# Card File Format

**go-mem** accepts plain text files as input. Each file can contain one or more "cards" (memory challenges).

## Basic Format

The simplest card is just a text file with content.

**Example `lorem.txt`:**
```text
Lorem ipsum dolor sit amet, consectetur adipiscing elit.
Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.
```

## Naming Cards

You can assign a custom title to a card by adding a `NAME:` line at the very beginning of the card content. This title will be displayed in the game banner and score history instead of the filename.

**Example `named_card.txt`:**
```text
NAME: My Custom Title
This is the text content you will type.
```

*   The `NAME:` line is **removed** from the playable text, so you don't need to type it.
*   The title is purely for display and organization.

## Multiple Cards in One File

You can define multiple cards in a single file by separating them with a line containing three or more dashes (`---`).

**Example `quotes.txt`:**
```text
NAME: Hamlet Quote
To be, or not to be, that is the question.
---
NAME: As You Like It
All the world's a stage,
And all the men and women merely players.
---
The only thing we have to fear is fear itself.
```

### Automatic Numbering
If a card in a multi-card file does **not** have a `NAME:` header, it will automatically be assigned a title based on the filename and its position in the file (e.g., `Quotes #3`).
