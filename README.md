# cdcq - A Cadence AST Query Tool

cdcq is a powerful command-line tool that allows you to query and analyze the Abstract Syntax Tree (AST) of your Cadence smart contracts. 

## Usage

```bash
cdcq <filename> <query>
```

### Examples

*List all functions in contract:*

```bash
➜  cdcq git:(main) ✗ ./cdcq ExampleToken.cdc ".Function | Access: {Function.Access} name: {Function.Identifier}"
```

Output:
```
Access: AccessPublic name: withdraw
Access: AccessPublic name: deposit
Access: AccessPublic name: withdraw
Access: AccessPublic name: deposit
Access: AccessPublic name: createEmptyVault
Access: AccessPublic name: mintTokens
```

*List all composites with CompositeKind Resource:*

   ```bash
   cdcq ExampleToken.cdc ".Composite[CompositeKind=~Resource] | {Composite.Identifier}"
   ```
  
Output:
```cadence
 Vault
 VaultMinter
```

*List resources and their conformances:*

   ```bash
   cdcq ExampleToken.cdc ".Composite[CompositeKind=~Resource] | {Composite.Identifier} {Composite.Conformances}" 
  ```

Output:
```cadence
 Vault [Provider Receiver Balance]
 VaultMinter []
```

*List variable declarations:*

   ```bash
   cdcq ExampleToken.cdc ".Variable | variable: {Variable}"     
   ```
Output:
```cadence
variable: let recipientRef = recipient.borrow() ?? panic("Could not borrow a receiver reference to the vault")
variable: let vault <- create Vault(balance: self.totalSupply)
```

## Running on Multiple Files

To analyze multiple Cadence files, use the `find` command:

```bash
find . -type f -name "*.cdc" -exec cdcq {} ".Variable | variable: {Variable}" \;
```

## Query Syntax

A cdcq query consists of a filter and a display section, separated by a pipe (`|`).

### Filter

* **Select:** Start with a period (`.`) followed by the AST element type (e.g., `.Function`, `.Composite`).
* **Choose:** Filter further using key-value pairs enclosed in square brackets (`[]`).
    * **Key:** A valid field within the AST element.
    * **Value:** The desired value to match. Prefix with `~` for "contains" matching.
    * **Supported Operations:** `=` (exact match), `!=` (not equal).

### Display

* A format string where variables are the AST element names from your filter, enclosed in curly braces (`{}`).

## Contributing

cdcq is an open-source project. We welcome contributions, issues, and feature requests. To get started, please refer to our contribution guidelines at [link to guidelines, if applicable].

