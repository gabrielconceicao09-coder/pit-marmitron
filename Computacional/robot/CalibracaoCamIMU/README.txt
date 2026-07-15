Calibração Câmera e IMU:


Parâmetros Intrínsecos MPU9250:

Captura de várias horas do sensor parado foi realizada com sketch arduino aquisicaoMPU. Lida e convertida em arquivo .csv pelo script serial_imu_to_csv.py.

Script params_imu_por_allan.py usa biblioteca allantools para calcular densidade de ruído e random walk do MPU9250 pela aquisição do sensor parado. CalibracaoMPU9250/imu_orbslam3.yaml contém os resultados.

Gravação da rotina de calibração é feita no workspace ros2 calibracao_ws, gravando os resultados em bag ros2. Bag ros2 é então convertida em rosbag ros1 com ferramenta CLI rosbags para uso com Kalibr.

Ferramenta de calibração Kalibr montada em docker. Pasta "para_montar" é montada no docker contendo a gravação e arquivos de calibração da IMU e do padrão usado. Rodando Kalibr, é possível calibrar a câmera, obtendo parâmetros de distorção e o conjunto câmera+IMU, obtendo diferença de tempo entre as medidas e matriz de transformação entre os eixos de referência de cada sensor.

Calibracao*_*_*/ contêm os resultados de algumas calibrações feitas sob diferentes condições.
