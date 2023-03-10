# fastplate
## Description
fastplate allows you to split any plain text file into multiple and combine them when it's required.  
Splitting files in partials has many possible benefits, one of them being to unclutter large files.  

## Setup
To use fastpalte, download the latest [release](https://github.com/xIRoXaSx/fastplate/releases) or clone and build the project locally.  
After that, split your files into templates and partials and run fastplate with the required [options](#cli-options).  
Fitting your requirements, the templating can be adapted to the structure shown in `interpreter/testdata/src` 
or can be customized to your liking.  

## CLI options
When using fastplate you can pass the listed arguments / options down below.  

| Argument        | Description                                                                                    |
|-----------------|------------------------------------------------------------------------------------------------|
| -in {FilePath}  | The input path to complete.                                                                    |
| -out {FilePath} | The output path for the completed template(s).                                                 |
| -var {FilePath} | The optional variable file path for unscoped variables.                                        |
| -no-stats       | Disable stats printing.                                                                        |
| -indent         | Enable indention. Spaces / tabs in front of `import` statements will be used for the partials. |
| -crlf           | Split and join contents by CRLF (\r\n) instead of LF (\n).                                     |

### Example
1. Complete template "tempalte.json":  
   `fastplate -in tempalte.json -out completed.json -var fastplate.var`

---

2. Complete multiple templates inside the directory "src":  
   `fastplate -in src/ -out dest/ -indent`

---

## Syntax
The syntax for various interpretations are shown in the table down below.  
The prefix `# fastplate` is always needed for fastplate's  interpretations and can also be used in form of `#fastplate`.  

| Syntax                  | Description                                                                                                        |
|-------------------------|--------------------------------------------------------------------------------------------------------------------|
| import {FilePath}       | Import a file into the current template / partial. Paths are always relational to the working dir.                 |
| var {Name} = {Value}    | Declare a scoped variable of the name `{Name}` and the value `{Value}`.                                            |
| ignore {start / end}    | Starts / ends a ignore block. Lines between these declarations will not be written to the output file.             |
| foreach [{{var1}}, ...] | Loops over each variable until it hits `foreachend`. Use `{{value}}` and `{{index}}` respectively inside the loop. |
| foreachend              | Signals the end of the foreach loop.                                                                               |

### Variables
Import variables can be declared and used from inside the template / partial file (= scoped / local) or 
via an additional variable file (default: `fastplate.var`), which variables can be used throughout every 
template / partial (= unscoped / global).  
The syntax to declare both variable types is the same.  
fastplate automatically looks for `fastplate.var` in the current working directory. If existing, you can use the unscoped 
variables without passing in the `-var` argument.

#### Function
You can use the following functions for any type of variable:

| Function name | Description                                                                 | Example              |
|---------------|-----------------------------------------------------------------------------|----------------------|
| lower()       | Prints the variable's value in lower case.                                  | `{{lower(varName)}}` |
| upper()       | Prints the variable's value in upper case.                                  | `{{upper(varName)}}` |
| cap()         | Prints the first letter of each word of the variable's value in upper case. | `{{cap(varName)}}`   |


### Example
1. Import `src/partials/world.txt` (which contains "World!") into the current template.
```text
Hello
   # fastplate import src/partials/world.txt
```

Result:
```text
Hello
   World!
```

---

2. Declare and use a scoped variable:
```text
# fastplate var world = World!
Hello
   {{world}}
```

Result:
```text
Hello
   World!
```

---

3. Declare and use an unscoped variable:  
   File fastplate.var:
```text
# fastplate var world = World!
```

Template:
```text
Hello
   {{world}}
```
Result:
```text
Hello
   World!
```

---

4. Use a foreach loop to iterate over specific variables (`[]` brackets are optional), may also be used with unscoped vars:
```text
Shopping list:
# fastplate var apples = Apples
# fastplate var oranges = Oranges
# fastplate var bananas = Bananas
# fastplate foreach [ {{apples}}, {{oranges}}, {{bananas}} ]
{{index}}.) 2x {{value}}
# fastplate foreachend
```

Result:
```text
Shopping list:
  0.) 2x Apples
  1.) 2x Oranges
  2.) 2x Bananas
```

---

5. Use functions:
```text
Shopping list:
  # fastplate var apples = APPLES
  # fastplate var oranges = oranges
  # fastplate var bananas = bananas
  2x {{lower(apples)}}
  2x {{upper(oranges)}}
  2x {{cap(bananas)}}
```

Result:
```text
Shopping list:
  2x apples
  2x ORANGES
  2x Bananas
```