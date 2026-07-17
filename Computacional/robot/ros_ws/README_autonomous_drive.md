# autonomous_drive

Este é o pacote principal do Marmitron em termos de autonomia. Contém a descrição do robô (URDF/xacro), os plugins de simulação no Gazebo, o nó que converte comando de velocidade em RPM das rodas, e os launch files que sobem o sistema, sendo a simulação ou o stack para o robô físico.

## Pacotes necessários:

Nav2:
- Componente principal para a locomoção autônoma;
- É um framework completo com múltiplos outros pacotes para navegação dentro do ROS2.

robot_localization:
- Utilizado para fundir os sensores e criar a transformada do corpo do robô para a odometria e o mapa.
- E

slam_toolbox:
- Um conjunto de ferramentas para criação de SLAM 2D;
- Usado na simulação para criar/fazer o mapeamento do ambiente.

Twist_Mux:
- Um Mux com grande variedade de customizações, aceitando diversas entradas com prioridades diferentes, inicialmente havia um mux criado pelo próprio grupo, porém, devido ao avanço do projeto, uma versão mais fácil de customizar se tornou necessária.

## O URDF

O Marmitron é um robô diferencial: duas rodas motrizes e duas de apoio (tipo caster/roda bova), para tal, o modelo virtual do robô conta com:

- Um LiDAR 2D, visto que para rodar o NAV2, é necessário a publicação de um tópico `/map`, que, no caso da simulação, é criado pelo pacote slam_toolbox a partir dos dados desse LiDAR (o funcionamento dele visa simular o que o OrbSLAM3 faria com uma câmera);
- IMU;
- GPS;
- 6 sensores ultrassônicos distribuídos pelo chassi (frente, trás, esquerda, direita e dois nas diagonais frontais).

Existem três esqueletos xacro, pois o robô físico e o simulado precisam de coisas diferentes: `marmitron_base.urdf.xacro` é apenas a estrutura, sem plugins (no robô real, quem lê os sensores são os ESP32). Já `marmitron_base_sim.urdf.xacro` inclui tudo que o Gazebo precisa para simular os sensores. O arquivo `marmitron_base_sim_camera.urdf.xacro` é uma cópia direta deste, mas possui um link de uma câmera efetivamente. Seria usado para simulações que tentassem usar o OrbSLAM3 junto ao Gazebo. 

**`robot_design.xacro`** define o esqueleto: um `base_link` sem geometria, dois volumes de chassi empilhados (`chassiB` embaixo, `chassiA` em cima), as rodas dianteiras acopladas aos motores, e duas rodas de apoio esféricas com atrito reduzido para deslizar com facilidade (fazendo o papel das rodas bobas).

`materials.xacro` define apenas as cores usadas no RViz/Gazebo. `inertials.xacro` contém as macros de inércia (esfera, caixa, cilindro), valores usados pelo Rviz2 e Gazebo, tanto para a simulação quanto para o pleno funcionamento do Nav2.

## Sensores

| Arquivo | Descrição |
|---|---|
| `lidar.xacro` | O `lazer_frame` — comportamente associado a de uma câmera, simulado como LiDAR |
| `imu.xacro` | IMU, fixado no `chassiB` |
| `gps.xacro` | Link do GPS, fixado no `base_link`, visto que para o gps a diferença entre o base_link e o local exato é mínima |
| `ultrasound.xacro` | Versão dos ultrassons sem plugin de simulação, usada no robô físico |
| `ultrasound_sim.xacro` | Versão simulada, com plugin de ray sensor, publicando `sensor_msgs/Range` |
| `camera.xacro` | Link de Câmera Monocular, com um plugin `camera_controller`, publicando `sensor_msgs/Image` | 

No xacro de simulação, os 6 ultrassons são instanciados via macro, cada um com posição e tópico próprios (`ultrasound/front`, `rear`, `left`, `right`, `fe`, `fd`).

## Controlador / plugins do Gazebo (Não confundir com o pid dos motores)

O `robot_controller.xacro` possui todos os plugins utilizados pelo Gazebo para simular sensores e as capacidades de um robô diferencial real:

- o LiDAR simulado, com 150 amostras e alcance de 0.3 a 12 m (a fim de se assemelhar mais ao funcionamento de uma câmera)
- o IMU, com ruído gaussiano configurado
- o diff_drive, que permite o controle do robô simulado e a leitura da posição de suas rodas para a odometria.
- o joint_state_publisher para montar a arvore de transformadas entre cada parte do robô.

Todos os tópicos são parametrizados sem prefixo fixo, permitindo remapeamento nos launch files.

## Nó de velocidade das rodas

`calculo_vel_rodas.cpp` escuta `/cmd_vel` e converte velocidade linear e angular em RPM de cada roda:

```
vel_dir = linear_x + (angular * R / 2)
vel_esq = linear_x - (angular * R / 2)
rpm     = (vel / (2π * r)) * 60
```

`R` (distância entre rodas) e `r` (raio da roda) estão fixos no código como `0.215` e `0.05` — os mesmos valores usados em `robot_controller.xacro`. Se as dimensões do robô mudarem, é preciso lembrar de atualizar. O nó publica o RPM de cada roda em `vel_roda_dir` e `vel_roda_esquerda`, que deve então ser passado para a ESP32, referente aos motores reais.

## Launch files

- **`rsp.launch.py`** — sobe o `robot_state_publisher`, processando o xacro (físico ou sim) conforme o argumento `URDF_file`.
- **`setup.launch.py`** — sobe a navegação (inclui o `navigation_launch.py` do Nav2), remapeando `cmd_vel` para `vel_nav` para passar pelo twist_mux. Aguarda 10s antes de subir, esse tempo é necessário para garantir que todos os demais nós estão funcionando como devido antes do início da publicação em `/map`.
- **`sim.launch.py`** — o launch completo de simulação: Gazebo, spawn do robô, RSP, slam_toolbox, os dois EKFs, NavSatFix, GPS, cálculo de velocidade, twist_mux, RViz, o nó de calibração de escala e, só depois disso tudo, o Nav2.
- **`full_launch.launch.py`** — a versão para o robô físico. Atualmente semi-incompleto, visto que, mesmo com todos os nós prontos para receberem os dados dos sensores (através do serial_comms, cujas mensagens foram adequadas ao que é necessário), seria preciso ter o OrbSLAM3 rodando para obter o mapa, além do tópico de odometria que esse deveria ter.


### EKF — `ekf_odom.yaml` e `ekf_map.yaml`

São os dois EKFs em série. O primeiro, funde apenas odometria das rodas, IMU e do proprio SLAM, sem GPS, usado para um processamento e fusão de sensores com alta taxa de amostragem, responsável pelo frame `odom`. O segundo funde a saída do primeiro com a odometria do GPS (via `navsat_transform_node`), sendo responsável pela transformada `map → odom`. O `navsat_transform` usa o yaw do EKF local em vez do gerado pelo próprio GPS e aguarda 3 segundos antes de começar a publicar, tal comportamente é padrão do robot_localization e pelo que pesquisei, não é recomendado mudar.

### SLAM — `mapper_params_online_async.yaml`

Configuração do SLAM Toolbox em modo mapping, lendo `/lidar`. Possui fechamento de loop habilitado, visto que esse, assim como testado, melhora em munto as capacidades do robô. Vale falar que, esse nó é apenas utilizado na simulação, pois assume o papel do SLAM, que normalmente seria gerado pelo OrbSLAM no modelo físico. Ou seja, não é um tópico existente durante uma movimentação real do robô.


### Twist mux — `twist_mux_topics.yaml`

Possui o papel de controlar quais mensagens são passadas para os motores, sendo que, mensagens vindas de fora (o controle pelo aplicativo de celular no caso) possuem maior prioridade, enquanto as mensagens de calibração possuem a segunda maior prioridade, e caso nenhuma dessas esteja sendo publicada, os comandos de velocidade do Nav2 são passados para as rodas.

### RViz — `config.rviz`

Layout salvo com o RobotModel, a grid e a lista de links do robô, apenas torna mais prático a execução das simulações, não muito importante fora isso.


## Fluxo de inicialização

No `sim.launch.py`, a ordem de inicialização é:

```
0s  → RSP, cálculo de velocidade, twist_mux, GPS, RViz e os dois EKFs
1s  → Gazebo
2s  → navsat_transform, spawn do robô
3s  → SLAM Toolbox
6s  → nó de calibração de escala
8s  → chamada bloqueante ao serviço de calibração, e só então o Nav2
```

Já no `full.launch.py`, a ordem de inicialização ficou como:

```
0s  → RSP, cálculo de velocidade, twist_mux e os dois EKFs
2s  → navsat_transform
6s  → nó de calibração de escala
6s ~ 30s  → setup (iniciado apenas quando a calibragem foi realizada)
```

Os atrasos foram programados pois, certos nós e partes precisam que as anteriores estejam pelnamente funcionando para iniciarem corretamente. Ao iniciar todos juntos poderiam ocorrer falhas, de maneira que a simulação ou o robô não funcionassem como devido


## Tipos de mensagens necessárias dos sensores

Mensagens utilizadas por cada sensores, e o que cada uma precisa.

## Header

Antes de detalhar as mensagens de cada sensor, é importante entender o campo `header`, já que esse é um campo obrigatorio para todas.

O `header` contém duas informações essenciais:

- **timestamp**: momento em que o dado foi medido.
- **frame_id**: nome exato de um link usado na árvore TF2 do robô. Esse nome pode ser encontrado na descrição URDF.

---

## Ultrassônicos

**Tipo:** `sensor_msgs/msg/Range`

| Campo | Descrição |
|---|---|
| `header` | Timestamp e frame ID |
| `uint8 radiation_type` | Tipo de radiação. Como usamos ultrassom, o valor é `0` |
| `float32 field_of_view` | Ângulo de abertura do sensor (radianos) |
| `float32 min_range` | Limite mínimo detectado pelo sensor (metros) |
| `float32 max_range` | Limite máximo detectado pelo sensor (metros) |
| `float32 range` | Distância até o objeto (metros). Pode retornar `-inf` ou `+inf` em caso de objetos muito próximos ou ausência de objeto, respectivamente |

---

## IMU

**Tipo:** `sensor_msgs/msg/Imu`

> **Observação 1:** A aceleração e a posição geradas pelo IMU geralmente são propensas a erro, por isso costumam não ser usadas no `robot_localization`. Para simplificar, esses campos podem ser zerados ao publicar a mensagem, mas é necessário avaliar o nosso caso.

> **Observação 2:** Campos que usam outros tipos de mensagem (como `Vector3`) podem ser consultados na documentação oficial. Porém, estão inclusos um exemplo de cada.

| Campo | Descrição |
|---|---|
| `header` | Timestamp e frame ID |
| `geometry_msgs/Quaternion orientation` | Orientação em quatérnions (`x`, `y`, `z`, `w`). *Verificar se o IMU calcula a orientação; caso não calcule, zerar todos os campos* |
| `float64[9] orientation_covariance` | Matriz 3x3 de covariância da orientação |
| `geometry_msgs/Vector3 angular_velocity` | Velocidade angular nos eixos `x`, `y`, `z` (rad/s) |
| `float64[9] angular_velocity_covariance` | Matriz 3x3 de covariância da velocidade angular |
| `geometry_msgs/Vector3 linear_acceleration` | Aceleração linear nos eixos `x`, `y`, `z` |
| `float64[9] linear_acceleration_covariance` | Matriz 3x3 de covariância da aceleração linear |

**Estrutura de `Vector3`**
|float64 x|
|float64 y|
|float64 z|
---

## GPS

**Tipo:** `sensor_msgs/msg/NavSatFix`

| Campo | Descrição |
|---|---|
| `header` | Timestamp e frame ID |
| `sensor_msgs/NavSatStatus status` | Ver detalhes abaixo |
| `float64 latitude` | Latitude em graus (negativo = sul) |
| `float64 longitude` | Longitude em graus (negativo = oeste) |
| `float64 altitude` | Altitude em metros |
| `float64[9] position_covariance` | Matriz 3x3 de covariância da posição (pode ser necessário definir manualmente) |

**Estrutura de `NavSatStatus`:**

| Campo | Descrição |
|---|---|
| `status.status` | Fornecido pelo próprio GPS |
| `status.service` | Componente decidido por nós, dependendo do modo que se deseja utilizar o gps |

> O nó `navsat` converte essa mensagem para o formato utilizado pelo `robot_localization`.

---

## Rodas

**Tipo:** `geometry_msgs/msg/TwistWithCovarianceStamped`

| Campo | Descrição |
|---|---|
| `header` | Timestamp e frame ID |
| `twist` | Do tipo `TwistWithCovariance`, que contém uma mensagem `geometry_msgs/msg/Twist` |

> O `twist` usa a velocidade linear e angular de ambas as rodas juntas.