#!/usr/bin/env python3

import math
import statistics
import time
import threading
from collections import deque

import rclpy
from rclpy.node import Node
from rclpy.callback_groups import MutuallyExclusiveCallbackGroup, ReentrantCallbackGroup
from rclpy.executors import MultiThreadedExecutor

from geometry_msgs.msg import Twist, PoseWithCovarianceStamped
from nav_msgs.msg import Odometry

from custom_msgs.srv import MsgMapScale


class Buffer:

    def __init__(self, tamanho_buffer: int, lim: float):

        #deque é uma classe especificamente eficiente para adicionar e tirar números das pontas
        #nesse caos é usada como um buffer que guarda até o escolhido
        self._buf = deque(maxlen=tamanho_buffer)

        #quantas medições serão tiradas
        self._tam_buff = tamanho_buffer

        #a diferença máxima entre cada medida (usado para ver se esta estavel)
        self._limite = lim

    def add(self, x: float, y: float):
        self._buf.append((x, y))

    def se_estavel(self) -> bool:
        if len(self._buf) < self._tam_buff:
            return False
        x_soma = [p[0] for p in self._buf]
        y_soma = [p[1] for p in self._buf]

        #nota: statistics.pstdev calcula automaticamente o desvio padrão dos npumeros passados
        return (statistics.pstdev(x_soma) < self._limite and
                statistics.pstdev(y_soma) < self._limite)

    #calcula a media das medidas do buffer
    def media(self):
        x_m = [i[0] for i in self._buf]
        y_m = [j[1] for j in self._buf]
        return (sum(x_m) / len(x_m), sum(y_m) / len(y_m))

    #Auto explicativo
    def has_data(self):
        return len(self._buf) > 0


class Calib_map(Node):

    def __init__(self):
        super().__init__('scale_calib_node')

        
        self.declare_parameter('ekf_odom_topic', 'odometry/filtered_map')
        self.declare_parameter('slam_pose_topic', 'odometry/filtered_map')
        self.declare_parameter('cmd_vel_topic', '/calib_vel')
        self.declare_parameter('vel_linear', 0.1)       
        self.declare_parameter('tempo_movimento', 5.0)        # s
        self.declare_parameter('num_repeats', 1)
        self.declare_parameter('tamanho_buffer', 10)   # numero de medidas
        self.declare_parameter('lim_estavel', 0.01)  # m
        self.declare_parameter('timeout_estabilizar', 3.0)  # s
        self.declare_parameter('espera_parar', 1.0)  # s
        self.declare_parameter('comandos_por_segundo', 20.0) # autoexplicativo
        self.declare_parameter('dist_min', 0.01)  # m, a distancia minima que o slam precisa dar


        self._shutdown_event = threading.Event()
        self._ekf_topic = self.get_parameter('ekf_odom_topic').value
        self._slam_topic = self.get_parameter('slam_pose_topic').value
        self._cmd_vel_topic = self.get_parameter('cmd_vel_topic').value
        self._controll_rate = self.get_parameter('comandos_por_segundo').value

        window = self.get_parameter('tamanho_buffer').value
        std_thr = self.get_parameter('lim_estavel').value

        #Precisa de 1 buffer para as medidas reais, e 1 para as medidas do orbslam
        self._ekf_buffer = Buffer(window, std_thr)
        self._orbslam_buffer = Buffer(window, std_thr)

        # Callback, 2 pois nada garante que ambos o slam e o ekf vão publicar ao mesmo tempo,
        # Logo precisa de 1 callback para os subscribers e 1 para o serviço em si
        self._service_cb = MutuallyExclusiveCallbackGroup()
        self._sub_cb = ReentrantCallbackGroup()


        # Publish topics, padrão

        # Vai enviar as msg para os motores
        self._cmd_vel_pub = self.create_publisher(Twist, self._cmd_vel_topic, 10)

        #fica ouvindo o ekf
        self.create_subscription(
            Odometry, self._ekf_topic, self._ekf_cb, 20,
            callback_group=self._sub_cb)

        #fica ouvindo o orbslam
        self.create_subscription(
            Odometry, self._slam_topic, self._slam_cb, 20,
            callback_group=self._sub_cb)

        # O Sercviço que o nó permite (srv = service)
        self._srv = self.create_service(
            MsgMapScale, 'calibrate_map_scale', self._handle_calib,
            callback_group=self._service_cb)

        self.get_logger().info(
            f'Escutando EKF topic="{self._ekf_topic}", '
            f'SLAM topic="{self._slam_topic}", cmd_vel="{self._cmd_vel_topic}"')


    # Definindo as funções que guardam as informações dos "subscribers"
    def _ekf_cb(self, msg: Odometry):
        p = msg.pose.pose.position
        self._ekf_buffer.add(p.x, p.y)


    def _slam_cb(self, msg: Odometry):
        p = msg.pose.pose.position
        self._orbslam_buffer.add(p.x, p.y)

    

    # A parte pricnipal:

    # Espera até as medições estarem estaveis durante o tempo determinado
    def _estavel(self, timeout: float):

        start = time.monotonic()
        sleep = 1.0 / max(self._controll_rate, 1.0) #espera por qual for maior

        while time.monotonic() - start < timeout:
            if self._ekf_buffer.se_estavel() and self._orbslam_buffer.se_estavel():
                return self._ekf_buffer.media(), self._orbslam_buffer.media(), True
            
            #caso um dos 2 buffers não esteja estavel ele espera mais um tempo
            time.sleep(sleep)

        # Em caso de dar o time out,
        self.get_logger().warn('Não foi possível estabilizar, usando os ultimso dados')
        
        #pega a media, caso não tenha medido nada por alugum motivo retorna 0
        ekf_xy = self._ekf_buffer.media() if self._ekf_buffer.has_data() else (0.0, 0.0)
        slam_xy = self._orbslam_buffer.media() if self._orbslam_buffer.has_data() else (0.0, 0.0)
        return ekf_xy, slam_xy, False


    #Função que gera as mensagens para os motores
    def _comando_vel(self, vel: float, tempo: float):
        twist = Twist()
        twist.linear.x = vel

        sleep = 1.0 / max(self._controll_rate, 1.0) #mesma coisa da função anterior
        start = time.monotonic()

        #anda pelo tempo determinado
        while (time.monotonic() - start) < tempo:
            self._cmd_vel_pub.publish(twist)
            time.sleep(sleep)

        #usado paar garantir que ele vai parar 
        stop = Twist()
        for i in range(3):
            self._cmd_vel_pub.publish(stop)
            time.sleep(0.05)

    # função que só faz a distancia entre pontos
    @staticmethod
    def _dist(a, b):
        return math.hypot(b[0] - a[0], b[1] - a[1])


    # A função que roda o processo, para ajudar em debug ela tem um nome
    def _start(self, vel: float, tempo: float, timeout: float,
                 espera: float, dist_min: float, nome: str):

        ekf_start, slam_start, estavel_start = self._estavel(timeout)
        self.get_logger().info(
            f'[{nome}] start: EKF={ekf_start} SLAM={slam_start} '
            f'(Resultado estabilidade={estavel_start})')

        self._comando_vel(vel, tempo)

        # Depois de andar espera um pouco para medir
        time.sleep(espera)

        ekf_end, slam_end, estavel_end = self._estavel(timeout)
        self.get_logger().info(
            f'[{nome}] end: EKF={ekf_end} SLAM={slam_end} '
            f'(Resultado estabilidade={estavel_end})')

        ekf_dist = self._dist(ekf_start, ekf_end)
        slam_dist = self._dist(slam_start, slam_end)

        if slam_dist < dist_min:
            self.get_logger().error(
                f'[{nome}] Distancia muito pequena')
            return None

        escala = ekf_dist / slam_dist
        return escala

    

    # Handler do serviço em si
    def _handle_calib(self, request, response):
        vel = request.linear_vel or self.get_parameter('vel_linear').value
        tempo = request.tempo or self.get_parameter('tempo_movimento').value
        repetir = request.n_repetir or self.get_parameter('num_repeats').value

        timeout = self.get_parameter('timeout_estabilizar').value
        espera = self.get_parameter('espera_parar').value
        min_dist = self.get_parameter('dist_min').value

        escalas = []

        try:
            for i in range(repetir):

                # faz a escala indo para frente
                escala_frente = self._start(
                    vel, tempo, timeout, espera,
                    min_dist, nome=f'ciclo{i+1}: frente')
                
                if escala_frente is not None:
                    escalas.append(escala_frente)

                # faz a escala indo para tras
                escala_tras = self._start(
                    -vel, tempo, timeout, espera,
                    min_dist, nome=f'ciclo{i+1}: tras')

                if escala_tras is not None:
                    escalas.append(escala_tras)

        except Exception as exc:  # em caso de erro queremos abortar o movimento, no minimo
            self.get_logger().error(f'Erro: {exc}')
            self._comando_vel(0.0, 0.0)  # para o robo
            response.sucesso = False
            response.mensagem = f'Falha: {exc}'
            response.escala = 0.0
            return response

        if not escalas:
            response.sucesso = False
            response.mensagem = ('Erro crítico')
            response.escala = 0.0
            return response

        media = statistics.mean(escalas)

        response.sucesso = True
        response.mensagem = ('Calibraçã ocompleta')
        response.escala = media

        #so para ficar mais arrumado
        if response.sucesso:
            self.get_logger().info(f'Termino calibração')
            threading.Thread(target=self._shutdown).start()

        return response

    # faz o no desligar após calibragem
    def _shutdown(self):
        time.sleep(0.5)
        self.get_logger().info('Calibragem realizada, Desligando')
        self._shutdown_event.set()

def main(args=None):
    rclpy.init(args=args)
    node = Calib_map()

    #precisa de mais de 1 thread devido aos callbacks
    executor = MultiThreadedExecutor(num_threads=4)
    executor.add_node(node)
    try:
        while rclpy.ok() and not node._shutdown_event.is_set():
            executor.spin_once(timeout_sec=0.1)
    except KeyboardInterrupt:
        pass
    finally:
        node.destroy_node()
        rclpy.shutdown()


if __name__ == '__main__':
    main()
