package cmd

import (
	"bufio"
	"bytes"
	"os"

	"github.com/spf13/cobra"
)

const boilerPlate = `
# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
`

func init() {
	autocompleteCmd.AddCommand(bashAutocompleteCmd)
	autocompleteCmd.AddCommand(zshAutocompleteCmd)

	RootCmd.AddCommand(autocompleteCmd)
}

var autocompleteCmd = &cobra.Command{
	Use:   "completion",
	Short: "Output shell completion code for the given shell (bash or zsh)",
}

var bashAutocompleteCmd = &cobra.Command{
	Use:   "bash",
	Short: "Output shell completion code for bash",
	Long: `
Output shell completion code for bash.
This command prints shell code which must be evaluated to provide interactive
completion of awless commands.
		$ source <(awless completion bash)
will load the awless completion code for bash. Note that this depends on the
bash-completion framework. It must be sourced before sourcing the awless
completion, e.g. on the Mac:
		$ brew install bash-completion
		$ source $(brew --prefix)/etc/bash_completion
		$ source <(awless completion bash)`,
	RunE: runCompletionBash,
}

var zshAutocompleteCmd = &cobra.Command{
	Use:   "zsh",
	Short: "List users",
	Long: `
Output shell completion code for zsh.
This command prints shell code which must be evaluated to provide interactive
completion of awless commands.
	$ source <(awless completion zsh)
[1] zsh completions are only supported in versions of zsh >= 5.2`,
	RunE: runCompletionZsh,
}

func runCompletionBash(cmd *cobra.Command, args []string) error {
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	_, err := out.Write([]byte(boilerPlate))
	exitOn(err)
	return RootCmd.GenBashCompletion(out)
}

func runCompletionZsh(cmd *cobra.Command, args []string) error {
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	_, err := out.Write([]byte(boilerPlate))
	exitOn(err)
	zshInitialization := `
__awless_bash_source() {
	alias shopt=':'
	alias _expand=_bash_expand
	alias _complete=_bash_comp
	emulate -L sh
	setopt kshglob noshglob braceexpand
	source "$@"
}
__awless_type() {
	# -t is not supported by zsh
	if [ "$1" == "-t" ]; then
		shift
		# fake Bash 4 to disable "complete -o nospace". Instead
		# "compopt +-o nospace" is used in the code to toggle trailing
		# spaces. We don't support that, but leave trailing spaces on
		# all the time
		if [ "$1" = "__awless_compopt" ]; then
			echo builtin
			return 0
		fi
	fi
	type "$@"
}
__awless_compgen() {
	local completions w
	completions=( $(compgen "$@") ) || return $?
	# filter by given word as prefix
	while [[ "$1" = -* && "$1" != -- ]]; do
		shift
		shift
	done
	if [[ "$1" == -- ]]; then
		shift
	fi
	for w in "${completions[@]}"; do
		if [[ "${w}" = "$1"* ]]; then
			echo "${w}"
		fi
	done
}
__awless_compopt() {
	true # don't do anything. Not supported by bashcompinit in zsh
}
__awless_declare() {
	if [ "$1" == "-F" ]; then
		whence -w "$@"
	else
		builtin declare "$@"
	fi
}
__awless_ltrim_colon_completions()
{
	if [[ "$1" == *:* && "$COMP_WORDBREAKS" == *:* ]]; then
		# Remove colon-word prefix from COMPREPLY items
		local colon_word=${1%${1##*:}}
		local i=${#COMPREPLY[*]}
		while [[ $((--i)) -ge 0 ]]; do
			COMPREPLY[$i]=${COMPREPLY[$i]#"$colon_word"}
		done
	fi
}
__awless_get_comp_words_by_ref() {
	cur="${COMP_WORDS[COMP_CWORD]}"
	prev="${COMP_WORDS[${COMP_CWORD}-1]}"
	words=("${COMP_WORDS[@]}")
	cword=("${COMP_CWORD[@]}")
}
__awless_filedir() {
	local RET OLD_IFS w qw
	__debug "_filedir $@ cur=$cur"
	if [[ "$1" = \~* ]]; then
		# somehow does not work. Maybe, zsh does not call this at all
		eval echo "$1"
		return 0
	fi
	OLD_IFS="$IFS"
	IFS=$'\n'
	if [ "$1" = "-d" ]; then
		shift
		RET=( $(compgen -d) )
	else
		RET=( $(compgen -f) )
	fi
	IFS="$OLD_IFS"
	IFS="," __debug "RET=${RET[@]} len=${#RET[@]}"
	for w in ${RET[@]}; do
		if [[ ! "${w}" = "${cur}"* ]]; then
			continue
		fi
		if eval "[[ \"\${w}\" = *.$1 || -d \"\${w}\" ]]"; then
			qw="$(__awless_quote "${w}")"
			if [ -d "${w}" ]; then
				COMPREPLY+=("${qw}/")
			else
				COMPREPLY+=("${qw}")
			fi
		fi
	done
}
__awless_quote() {
    if [[ $1 == \'* || $1 == \"* ]]; then
        # Leave out first character
        printf %q "${1:1}"
    else
    	printf %q "$1"
    fi
}
autoload -U +X bashcompinit && bashcompinit
# use word boundary patterns for BSD or GNU sed
LWORD='[[:<:]]'
RWORD='[[:>:]]'
if sed --help 2>&1 | grep -q GNU; then
	LWORD='\<'
	RWORD='\>'
fi
__awless_convert_bash_to_zsh() {
	sed \
	-e 's/declare -F/whence -w/' \
	-e 's/local \([a-zA-Z0-9_]*\)=/local \1; \1=/' \
	-e 's/flags+=("\(--.*\)=")/flags+=("\1"); two_word_flags+=("\1")/' \
	-e 's/must_have_one_flag+=("\(--.*\)=")/must_have_one_flag+=("\1")/' \
	-e "s/${LWORD}_filedir${RWORD}/__awless_filedir/g" \
	-e "s/${LWORD}_get_comp_words_by_ref${RWORD}/__awless_get_comp_words_by_ref/g" \
	-e "s/${LWORD}__ltrim_colon_completions${RWORD}/__awless_ltrim_colon_completions/g" \
	-e "s/${LWORD}compgen${RWORD}/__awless_compgen/g" \
	-e "s/${LWORD}compopt${RWORD}/__awless_compopt/g" \
	-e "s/${LWORD}declare${RWORD}/__awless_declare/g" \
	-e "s/\\\$(type${RWORD}/\$(__awless_type/g" \
	<<'BASH_COMPLETION_EOF'
`
	out.Write([]byte(zshInitialization))

	buf := new(bytes.Buffer)
	RootCmd.GenBashCompletion(buf)
	out.Write(buf.Bytes())

	zshTail := `
BASH_COMPLETION_EOF
}
__awless_bash_source <(__awless_convert_bash_to_zsh)
`
	out.Write([]byte(zshTail))
	return nil
}
