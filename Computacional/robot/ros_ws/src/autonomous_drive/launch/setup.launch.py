import os
from ament_index_python.packages import get_package_share_directory

from launch_ros.actions import Node
from launch import LaunchDescription
from launch_ros.actions import SetRemap
from launch.substitutions import LaunchConfiguration
from launch.actions import DeclareLaunchArgument, TimerAction
from launch.actions import GroupAction, IncludeLaunchDescription
from launch.launch_description_sources import PythonLaunchDescriptionSource


def generate_launch_description():
    # Setando parametros e definindo caminhos
    Package_Name_self = "autonomous_drive"
    Package_share_dir = get_package_share_directory(Package_Name_self)
    nav2_bringup_dir = get_package_share_directory('nav2_bringup')

    Launcher_Nav2 = os.path.join(nav2_bringup_dir,  'launch', 'navigation_launch.py')

    param_file_nav2 = os.path.join(Package_share_dir, 'configs', 'nav2_params.yaml')

    use_sim_time = LaunchConfiguration('use_sim_time')


    # Definindo os executaveis e nós a serem lançados
    
    Nav2_launch = GroupAction([
    SetRemap(src='cmd_vel', dst='vel_nav'),
    IncludeLaunchDescription(
        PythonLaunchDescriptionSource(Launcher_Nav2),
        launch_arguments={
            'use_sim_time': use_sim_time,
            'params_file': param_file_nav2,
        }.items())
    ])


    return LaunchDescription([
        DeclareLaunchArgument(name='use_sim_time', default_value='true', description='Use sim time if true'),
        TimerAction(period=10.0, actions=[Nav2_launch]),   
    ])