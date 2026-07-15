import os
import rclpy
from rclpy.node import Node as RclpyNode
from ament_index_python.packages import get_package_share_directory

from launch_ros.actions import Node
from launch import LaunchDescription
from launch.actions import TimerAction, OpaqueFunction
from launch.substitutions import LaunchConfiguration
from launch.actions import IncludeLaunchDescription
from launch.actions import DeclareLaunchArgument
from launch.launch_description_sources import PythonLaunchDescriptionSource
from custom_msgs.srv import MsgMapScale

# Esse launch inicia o gazebo junto da bridge, para assim se visualizar a simulação


def _call_calib():
    rclpy.init(args=None)
    node = RclpyNode('map_scale_launch_client')
    client = node.create_client(MsgMapScale, 'calibrate_map_scale')

    if not client.wait_for_service(timeout_sec=30.0):
        rclpy.shutdown()
        raise RuntimeError('Timeout na calibragem')

    req = MsgMapScale.Request()
    future = client.call_async(req)
    rclpy.spin_until_future_complete(node, future, timeout_sec=600.0)

    if not future.done() or future.result() is None:
        rclpy.shutdown()
        raise RuntimeError('Timeout na calibragem')

    result = future.result()
    node.destroy_node()
    rclpy.shutdown()

    if not result.sucesso:
        raise RuntimeError(f'Falha em calibragem: {result.mensagem}')

    return result.escala


def _launch_setup(context, *args, **kwargs):
    package_share = get_package_share_directory('autonomous_drive')

    try:
        _call_calib()

    except Exception as e:
        while True:
            print("shit", e)

    setup = IncludeLaunchDescription(
        PythonLaunchDescriptionSource(os.path.join(package_share, 'launch', 'setup.launch.py')),
        launch_arguments={'use_sim_time': 'False'}.items()
    )

    return [setup]


def generate_launch_description():

    package_name_self = 'autonomous_drive'
    package_name_2 = 'map_scale_calib'
    package_share = get_package_share_directory(package_name_self)
    use_sim_time = LaunchConfiguration('use_sim_time')

    params_ekf_map = [os.path.join(package_share, "configs", "ekf_map.yaml"), {'use_sim_time': use_sim_time}]
    params_ekf_odom = [os.path.join(package_share, "configs", "ekf_odom.yaml"), {'use_sim_time': use_sim_time}]
    params_mux = [os.path.join(package_share, 'configs', 'twist_mux_topics.yaml'), {'use_sim_time': use_sim_time}]
    

    rsp = IncludeLaunchDescription(
        PythonLaunchDescriptionSource(os.path.join(package_share, 'launch', 'rsp.launch.py')),
        launch_arguments={'use_sim_time': 'false', 'URDF_file': 'marmitron_base.urdf.xacro'}.items()
    )


    # NÃO IRA FUNCIONAR, PORQUE O SLAMTOOLBOX NÃO GERA A POSIÇÃO E O EKF NÃO POSSUI O MEMSO TIPO DE MENSAGEM
    calib_node = Node(
        package=package_name_2,
        executable='node_calib',
        name='map_scale_calib',
        output='screen',
        emulate_tty=True,
        parameters=[{
        'use_sim_time': use_sim_time,
        'ekf_odom_topic': 'odometry/filtered_map',}],
    )

    calc_vel_node = Node(
        package=package_name_self,
        executable='vel_rodas',
        output='screen',
        name='Vel_rodas',
    )

    twist_mux_node = Node(
        package='twist_mux',
        executable='twist_mux',
        name='twist_mux',
        output='screen',
        parameters=params_mux,
        remappings=[('cmd_vel_out', 'cmd_vel')]
    )

    ekf_node_odom = Node(
        package='robot_localization',
        executable='ekf_node',
        name='ekf_filter_node',
        output='screen',
        parameters=params_ekf_odom,
        remappings=[('odometry/filtered', 'odometry/filtered_map')]
    )

    ekf_node_map = Node(
        package='robot_localization',
        executable='ekf_node',
        name='ekf_filter_map_node',
        output='screen',
        parameters=params_ekf_map,
        remappings=[('odometry/filtered', 'odometry/global')]
    )

    navsat_node = Node(
        package='robot_localization',
        executable='navsat_transform_node',
        name='navsat_transform',
        output='screen',
        parameters=params_ekf_map,
        remappings=[
            ('/gps/fix', '/gps/fix'),
            ('/odometry/filtered', '/odometry/filtered_map'),
            ('/odometry/gps', '/odometry/gps')
        ]
    )

    return LaunchDescription([
        DeclareLaunchArgument(name='use_sim_time', default_value='false', description='Use sim time if true'),
        rsp,
        calc_vel_node,
        calib_node,
        twist_mux_node,
        ekf_node_map,
        ekf_node_odom,
        TimerAction(period=2.0, actions=[navsat_node]),
        TimerAction(period=4.0, actions=[OpaqueFunction(function=_launch_setup)]),
    ])