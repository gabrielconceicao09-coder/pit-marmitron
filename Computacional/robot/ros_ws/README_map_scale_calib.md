# map_scale_calib

Pacote ROS 2 responsável pela calibragem automaticamente a escala do mapa gerado pelo SLAM adotado, pois devido a problemas técnicos esse não produz um mapa em escala real.

Motivo da existencia do pacote: o OrbSlam gera um mapa em uma escala que não corresponde necessariamente a metros reais. Para solucionar tal problema, o robô iria se desloca um pouco para frente e para trás em linha reta, mediria a distância percorrida segundo o EKF (em metros reais) e segundo o OrbSlam (escala imaginaria do SLAM), e então, a razão entre essas duas distâncias forneceria o fator de escala que, ao ser utilziado como 'resolution' permitira adequar o mapa para a escala real.

O seu funcionamento necessita de 2 pacotes:

| Pacote | Função |
|---|---|
| `map_scale_calib` | O próprio nó que executa a rotina de calibração |
| `custom_msgs` | Define o tipo de mensagem `MsgMapScale.srv` usado pelo nó |


## Idéia

O nó (`node_calib`) executa as seguintes funções ao receber uma chamada no serviço `calibrate_map_scale`:

1. Aguarda as leituras estabilizarem — as últimas posições vindas do EKF e do SLAM são acumuladas em dois buffers, e cada um é considerado estável quando o desvio padrão das últimas N medições fica abaixo de um limite configurável. Isso evita medir a posição enquanto o robô ainda está ruidoso.

2. Registra a posição inicial e movimenta o robô para frente por um tempo escolhido (`Twist` publicado em `/calib_vel` que deve passar por um Twist_Mux antes de chegar aso motores).

3. Aguarda estabilizar novamente e registra a posição final.

4. Calcula a razão entre as medidas: `distância segundo o EKF / distância segundo o SLAM`.

5. Repete o mesmo processo indo para trás, obtendo mais uma amostra.

6. Se solicitado, repete o ciclo completo (frente + trás) `n_repetir` vezes, retornando ao final a média das escalas válidas. (`n_repetir` é o nome do parametro que define quantas vezes esse comportamente aconetce)

7. Responde o serviço com sucesso/falha, uma mensagem e o valor calculado. (a mensagem serve para o cliente que pedir o serviço tenha um feedback melhor)

8. Ao concluir uma calibração com sucesso, o nó se desliga automaticamente (em uma thread separada), devido a isso, foi decidido utilziar python, pois após terminar a tarefa, ele não consumiria memória ou processamento.

Caso a distância medida pelo SLAM fique abaixo de `dist_min`, a medição é descartada, evitando divisão por valores muito pequenos ou ruidosos (devido a baixa resolução dos encoders).

O nó usa dois callback separados: um `Reentrant` para os subscribers do EKF/SLAM e outro `MutuallyExclusive` para o serviço em si. Isso é necessário porque o serviço fica bloqueado durante boa parte da execução (esperando estabilização, movendo o robô) e, sem essa separação, os subscribers ficariam impedidos de atualizar os buffers enquanto o serviço não termina.

## Tipo de mensagem gerada (referente ao pacote custom_msgs)

`calibrate_map_scale`, tipo `custom_msgs/srv/MsgMapScale`.

**Request**
| Campo | Tipo | Descrição |
|---|---|---|
| `linear_vel` | `float64` | Velocidade da calibração (m/s). Se `0`, usa o parâmetro `vel_linear` |
| `tempo` | `float64` | Duração do movimento (s). Se `0`, usa `tempo_movimento` |
| `n_repetir` | `int32` | Repetições do ciclo frente/trás. Se `0`, usa `num_repeats` |

**Response**
| Campo | Tipo | Descrição |
|---|---|---|
| `sucesso` | `bool` | Indica se a calibração foi concluída com sucesso |
| `mensagem` | `string` | Descrição do resultado (ou do erro, em caso de falha) |
| `escala` | `float64` | Fator de escala calculado — `0.0` em caso de falha |


## Parâmetros


| Parâmetro | Padrão | Unidade | Descrição |
|---|---|---|---|
| `ekf_odom_topic` | `odometry/filtered_map` | — | Tópico de odometria do EKF |
| `slam_pose_topic` | `odometry/filtered_map` | — | Tópico de pose do SLAM |
| `cmd_vel_topic` | `/calib_vel` | — | Tópico onde os comandos de velocidade são publicados |
| `vel_linear` | `0.1` | m/s | Velocidade padrão da calibração |
| `tempo_movimento` | `5.0` | s | Duração padrão de cada trecho de movimento |
| `num_repeats` | `1` | — | Número padrão de repetições do ciclo frente/trás |
| `tamanho_buffer` | `10` | amostras | Janela usada para checar estabilidade |
| `lim_estavel` | `0.01` | m | Desvio padrão máximo aceito como estável |
| `timeout_estabilizar` | `3.0` | s | Tempo máximo de espera pela estabilização |
| `espera_parar` | `1.0` | s | Espera após o robô parar, antes de medir a posição final |
| `comandos_por_segundo` | `20.0` | Hz | Frequência de publicação e checagem de estabilidade |
| `dist_min` | `0.01` | m | Distância mínima (segundo o SLAM) para a medição ser considerada válida |

Vale notar que `ekf_odom_topic` e `slam_pose_topic` apontam para o mesmo tópico por padrão (`odometry/filtered_map`), pois, devido a falta de tempo, o robô real em que esse serviço seria necessário não podê ser terminado a tempo.

## Observações

- O cálculo assume que o robô se move em linha reta pura, o que, em grande maioria dos casos, não ocorreria, porém, devido ao tempo e os recursos que possuiamos, essa acabou por ser a melhor solução obtida, ademais, caso o movimento seja em sua grande parte uma "linha", esse método deve ser preciso o bastante.
- `ekf_odom_topic` e `slam_pose_topic` compartilham o mesmo valor padrão então é necesssário alterar algum desses valores paar o robô real.
- Em caso de erro durante o movimento, o nó tenta parar o robô antes de retornar a falha no serviço.
- Se nenhuma medição for considerada válida (por exemplo, todas com distância do SLAM abaixo de `dist_min`), o serviço retorna `sucesso: false` com uma mensagem de erro crítico.
