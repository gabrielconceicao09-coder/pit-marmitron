# Roteiro de Apresentacao - MARMITRON 3000

## Abertura - 45 segundos

"O MARMITRON 3000 e uma plataforma de entrega autonoma. Nossa equipe construiu
a camada de produto: aplicativo, gateway em nuvem, persistencia, contrato MQTT,
observabilidade do robo e interface fisica no display. O sistema foi desenhado
para integrar ROS 2/Nav2 sem acoplar o app aos detalhes internos da navegacao."

Mostre a GitHub Release e o APK.

## 1. Aplicativo e jornadas - 2 minutos

Mostre os tres papeis: cliente, restaurante e operador.

- Cliente: catalogo, carrinho, ponto de entrega, acompanhamento e retirada por
  QR/OTP.
- Restaurante: pedidos e produtos.
- Operador: estado, telemetria, mapa local, camera preparada e E-stop.

Justificativa: papeis separados reduzem confusao operacional e mantem a tela do
operador focada em seguranca e observabilidade.

## 2. Pedido ate o robo - 1 minuto

"Ao despachar um pedido, o gateway publica um comando MQTT. O pedido carrega
route_id e nos de rota ordenados. O app nao calcula caminho livre: a rota
segura e um dado calibrado e auditavel."

Justificativa: entregar apenas latitude/longitude final faria o robo escolher
um caminho potencialmente inseguro. A responsabilidade e separada: produto
seleciona rota aprovada; ROS/Nav2 executa movimento e desvio local.

## 3. Gateway, banco e cloud - 1 minuto

Mostre a API e, se conveniente, o endpoint /health.

- Go recebe pedidos e publica comandos.
- PostgreSQL persiste catalogo, pedidos, pontos de entrega, rotas e telemetria.
- MQTT conecta nuvem, edge daemon e ESP32.
- A telemetria e enfileirada para o banco; o callback MQTT nao faz I/O pesado.

Justificativa: a fila protege o recebimento de mensagens e separa o snapshot
rapido usado pela interface do historico duravel usado para auditoria.

## 4. ROS, edge daemon e mapa - 2 minutos

"O mapa mostrado e um mapa ROS local no frame map, nao um mapa geografico.
A seta e a pose atual x/y/theta. A linha verde e a trilha executada: pontos
de pose recebidos pela telemetria e conectados em sequencia."

Execute o comando demo e mostre a seta e a trilha.

"A linha nao e a rota planejada. Ela deve terminar na seta; pode parecer fixa
porque a visualizacao reescala o conjunto inteiro de poses para caber na tela.
Na proxima evolucao, mostraremos rota planejada tracejada e trilha executada
continua, em cores diferentes."

Justificativa: separar plano de rota de evidencia de execucao evita confundir
intencao do planejador com a trajetoria realmente observada.

## 5. Seguranca operacional - 1 minuto

Mostre o botao de parada de emergencia e explique a confirmacao.

- O E-stop publica MQTT com QoS 2.
- O edge daemon cancela a meta Nav2 ativa.
- A interface explicita erro de entrega do comando quando o broker falha.
- Dados de telemetria vencem em 15 segundos; o painel nao pode afirmar que o
  robo esta em movimento com um snapshot antigo.

Justificativa: em sistemas fisicos, dado antigo e pior que ausencia de dado.

## 6. Display ESP32 - 1 minuto

Mostre o Wokwi ou o display fisico durante o comando demo.

- Tela de espera animada.
- Painel de missao com destino, progresso, distancia restante e estados.
- QR e OTP para retirada.
- Desbloqueio apenas apos QR correspondente; o firmware rejeita sequencias
  invalidas.

Justificativa: Wi-Fi, MQTT e desenho SPI rodam fora da tarefa PID do motor,
preservando o controle de movimento. O display compartilha apenas os contratos
MQTT, nao a logica critica do controlador.

## 7. O que foi validado - 45 segundos

- Pedido publica comando de navegacao com rota simulada.
- QR, desbloqueio, E-stop, telemetria, status de navegacao e display foram
  exercitados via MQTT.
- APK Android, nome e icone do MARMITRON 3000 foram gerados.
- O simulador publica pose progressiva e atualiza operador e display sem mover
  o robo.

## 8. Limites assumidos e proximos passos - 1 minuto

"Nao apresentamos como concluido o que depende de hardware e calibracao."

- Equipe de Computacao ainda deve entregar pontos seguros e validar frames,
  datum GPS, Nav2 e execucao fisica.
- Video da C920 depende do topico ROS e da URL MediaMTX/go2rtc. O contrato de
  camera e a tela do operador ja estao preparados.
- A demonstracao MQTT e segura, mas nao substitui validacao fisica.
- Autenticacao completa foi postergada por prioridade do projeto; credenciais,
  banco publico e hardening devem ser tratados antes de producao.

## Fechamento - 20 segundos

"O resultado e uma prova de conceito integrada por contratos: hoje
demonstramos a experiencia completa com simulacao; amanha, os mesmos contratos
permitem conectar o ROS, a camera e o robo fisico sem redesenhar o aplicativo."
