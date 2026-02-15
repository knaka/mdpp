# Convert JSON to shell variable declarations
#
# Usage: jq -r --from-file json2sh.jq [--arg prefix "my_prefix"] [--arg delim "__"] [--arg local_decl "local "]
#
# Parameters:
#   prefix:     Variable name prefix (default: "json__")
#   delim:      Delimiter for nested keys (default: "__")
#   local_decl: Declaration prefix, e.g. "local " or "export " (default: "")
#
# Exampl--from-file
#   echo '{"user":{"name":"Alice","age":30},"items":["apple","banana"]}' | jq -r -f json2sh.jq
#   # Output:
#   # json__user__name="Alice"
#   # json__user__age="30"
#   # json__items__0="apple"
#   # json__items__1="banana"

# Recursively convert JSON object/array to shell variable declarations
def to_sh(prefix):
  to_entries[]
  | $ARGS.named.delim // "__" as $delim
  | $ARGS.named.local_decl // "" as $local_decl
  | (
      # Convert key to shell-safe identifier
      if .key | type == "number" then 
        .key | tostring
      else
        # Replace hyphens and dots with underscores
        .key | gsub("[-\\.]"; "_")
      end
    ) as $shell_key
  | if .value | type == "object" or type == "array" then
      # Recursively process nested objects/arrays
      .value | to_sh("\(prefix)\($shell_key)\($delim)")
    else
      # Generate shell variable declaration for primitive values
      "\($local_decl)\(prefix)\($shell_key)=\"\(.value)\""
    end
;

.
| $ARGS.named.prefix // "json__" as $prefix
| to_sh($prefix)
