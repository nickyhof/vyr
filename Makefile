.PHONY: test clean bootstrap docs

# Bootstrap: rebuild vyrc binary and update the checked-in Go source
bootstrap:
	@echo "Merging bootstrap files..."
	@python3 -c "\
	files = ['bootstrap/lexer.vyr','bootstrap/parser.vyr','bootstrap/codegen.vyr','bootstrap/main.vyr']; \
	lines = []; \
	[lines.extend(l for l in open(f) if not l.strip().startswith('import ')) or lines.append('\n') for f in files]; \
	open('/tmp/_vyr_merged.vyr','w').writelines(lines)"
	@echo "Compiling merged bootstrap with existing vyrc..."
	./vyrc /tmp/_vyr_merged.vyr bootstrap/vyrc.go
	go build -o vyrc bootstrap/vyrc.go
	@rm -f /tmp/_vyr_merged.vyr
	@echo "vyrc rebuilt. bootstrap/vyrc.go updated."

test:
	@echo "=== Compiling and running examples ==="
	@for f in examples/hello.vyr examples/structs.vyr; do \
		echo "Testing $$f..."; \
		tmpdir=$$(mktemp -d); \
		./vyrc $$f $$tmpdir/main.go && go run $$tmpdir/main.go || exit 1; \
		rm -rf $$tmpdir; \
	done
	@echo "All tests passed."

clean:
	rm -f vyrc

install-ext:
	ln -sfn $(CURDIR)/editors/vscode ~/.vscode/extensions/nickyhof.vyr-0.1.0
	@echo "Installed to VS Code."
	@if [ -d ~/.antigravity/extensions ]; then \
		ln -sfn $(CURDIR)/editors/vscode ~/.antigravity/extensions/nickyhof.vyr-0.1.0; \
		echo "Installed to Antigravity."; \
	fi
	@echo "Reload your editor to activate."

docs:
	@python3 scripts/gendocs.py
