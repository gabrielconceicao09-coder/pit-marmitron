# ros_ws

Workspace ROS 2 usado nas simulações de navegação do Marmitron. É aqui que ficam os pacotes voltados à simulação do projeto e os arquivos para a navegação autonoma do robô.


## Pacotes

| Pacote | Função | Documentação |
|---|---|---|
| `autonomous_drive` | Descrição do robô, plugins de simulação e launch files | [`README_autonomous_drive.md`](./README_autonomous_drive.md) |
| `custom_msgs` | Define a interface de serviço `MsgMapScale.srv` | [`README_map_scale_calib.md`](./README_map_scale_calib.md) |
| `map_scale_calib` | Calibra a escala do mapa do SLAM comparando com o EKF | [`README_map_scale_calib.md`](./README_map_scale_calib.md) |


## Dependências

Pacotes necessários para pleno funcionamento:

- `gazebo_ros`
- `slam_toolbox`
- `robot_localization`
- `Nav2`
- `twist_mux`


## Subindo a simulação

O ponto de entrada é o `sim.launch.py` do `autonomous_drive`.

Para se saber mais sobre o processo de inicializar a simulação, veja o readme do autonomous_drive.


## Notas

- A calibração de escala (`map_scale_calib`) é chamada de forma bloqueante durante o launch, de forma que o Nav2 só começa a funcionar depois que o serviço `calibrate_map_scale` responder. Se o EKF ou o SLAM não estiverem publicando corretamente, o launch IRÁ travar.

- `full_launch.launch.py` (pensado para o robô físico) incompleto, devido a limites de tempo.

- Os arquivos de configuração (EKF, SLAM, Nav2, twist_mux) ficam dentro de `autonomous_drive/configs/` e são os pricnipais arquivos a se mexar para mudar o comportamento do robô ou tentar melhora-lo.

