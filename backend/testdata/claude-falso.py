#!/usr/bin/env python3
# Claude falso para o teste de ponta a ponta da Rotina: responde como o "claude -p --output-format json
# --json-schema", com o structured_output. Entender a fala → um evento na próxima quinta, 19h–22h;
# sugerir → uma semana pequena.
import json, re, sys

args = sys.argv[1:]
pedido = args[args.index("-p") + 1] if "-p" in args else ""

if "Pedido:" in pedido:
    quinta = re.search(r"^(\d{4}-\d{2}-\d{2}) quinta$", pedido, re.M).group(1)
    saida = {"eventos": [{"data": quinta, "inicio": "07:00", "fim": "10:00", "titulo": "Encontro com amigos", "atividade": "", "nota": ""}],
             "observacao": "Entendi das 7h às 10h da manhã."}
elif "Atividades (id, nome" in pedido:
    prog = 3 if "programação" in pedido.lower() and "Pedido do Lucas" in pedido else 1
    blocos = [{"dia": d, "inicio": "07:00", "fim": "08:00", "atividade": "ingles", "nota": ""} for d in range(7)]
    blocos += [{"dia": d, "inicio": "20:00", "fim": "21:30", "atividade": "programacao", "nota": ""} for d in range(prog)]
    saida = {"explicacao": f"Inglês todo dia e programação {prog}x.", "blocos": blocos}
else:
    saida = None

print(json.dumps({"type": "result", "result": "", "is_error": saida is None, "structured_output": saida}))
