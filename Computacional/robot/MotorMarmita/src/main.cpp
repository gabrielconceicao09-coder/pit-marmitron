#include <Arduino.h>
#include "pinout.h"
#include "config.h"

#include "encoder_module.h"
#include "motor_module.h"
#include "ControlePID.h"

#include "webserver_module.h" 
#include "lock_service.h"


motor motorEsq(PIN_PWMR_ESQ, PIN_PWML_ESQ);
motor motorDir(PIN_PWMR_DIR, PIN_PWML_DIR);

PIDController pidEsq(KPesq, KIesq, KDesq); //0.9 0.1 0.0
PIDController pidDir(KPdir, KIdir, KDdir);

EncoderISR encEsq(PIN_ENC_ESQ, 0.95);
EncoderISR encDir(PIN_ENC_DIR, 0.95);

// ==========================================
// COMUNICAÇÃO ENTRE TAREFAS (FreeRTOS)
// ==========================================
// Criamos uma "caixa de correio" para a velocidade.
QueueHandle_t filaVelocidadeEsq;
QueueHandle_t filaVelocidadeDir;
QueueHandle_t filaComandosSerial;

// Identificadores das tarefas
TaskHandle_t TaskEncoders;
TaskHandle_t TaskMotores;
TaskHandle_t TaskInterface;
TaskHandle_t TaskSerialCom;

static LockService lockService;

// ==========================================
// TAREFA 0: COMUNICAÇÃO SERIAL
// ==========================================
void codigoTaskSerial(void * parameter) {
  char buffer[64];
  uint8_t idx = 0;
  
  for(;;) {
    // Processa dados disponíveis na Serial
    while (Serial.available() > 0) {
      char c = Serial.read();
      
      // Detecta fim de linha (Enter)
      if (c == '\n' || c == '\r') {
        if (idx > 0) {
          buffer[idx] = '\0'; // Finaliza string
          
          // Converte para float e envia para fila
          float comando = atof(buffer);
          
          // Envia comando para fila (usando timeout curto)
          if (xQueueSend(filaComandosSerial, &comando, 0) == pdTRUE) {
            Serial.print("[SERIAL] Comando recebido: ");
            Serial.println(comando);
          } else {
            Serial.println("[SERIAL] Erro: Fila cheia!");
          }
          
          idx = 0; // Reinicia buffer
        }
      } else if (idx < sizeof(buffer) - 1) {
        buffer[idx++] = c; // Armazena caractere
      }
    }
    
    vTaskDelay(pdMS_TO_TICKS(10)); // Pequeno delay para não sobrecarregar
  }
}

// ==========================================
// TAREFA 1: MOTORES & PID (Rodando no CORE 1)
// ==========================================
void codigoTaskMotores(void * parameter) {
  motorEsq.init();
  motorDir.init();
  encEsq.init();
  encDir.init();

  float vel_ref_esq = 0.0;
  float vel_ref_dir = 0.0; 
  float rpm_esq = 0.0;
  float rpm_dir = 0.0;
  const float BASE_RPM = 200.0;
  
  float comando_serial;
  float comando_web;

  for(;;) {
    bool comando_web_recebido = false;
    while (xQueueReceive(filaVelocidadeEsq, &comando_web, 0) == pdTRUE) {
      comando_web_recebido = true;
    }
    bool comando_serial_recebido = false;
    while (xQueueReceive(filaComandosSerial, &comando_serial, 0) == pdTRUE) {
      comando_serial_recebido = true;
    }
    if (comando_serial_recebido) {
      processarComando(comando_serial, vel_ref_esq, vel_ref_dir, BASE_RPM);
    } else if (comando_web_recebido) {
      processarComando(comando_web, vel_ref_esq, vel_ref_dir, BASE_RPM);
    }

    // 2. Lê a velocidade real atual medida pelos encoders
    float rpm_esq = encEsq.lerVelocidadeRPM(); 
    float rpm_dir = encDir.lerVelocidadeRPM();

    //Printa leituras encoders para uso na odometria
    Serial.print(encEsq.timestamp); Serial.print(",");
    Serial.print(encEsq.sequencia); Serial.print(",");
    encEsq.printLeitura(vel_ref_esq); Serial.print(",");
    Serial.print(encDir.timestamp); Serial.print(",");
    Serial.print(encDir.sequencia); Serial.print(",");
    encDir.printLeitura(vel_ref_dir); Serial.println();

    // Atualiza o sentido com base no sinal da referência
    pidEsq.setSentido(vel_ref_esq);
    pidDir.setSentido(vel_ref_dir);

    // 3. O PID calcula o valor de PWM necessário
    int pwm_esq = pidEsq.controle(abs(vel_ref_esq), rpm_esq);
    int pwm_dir = pidDir.controle(abs(vel_ref_dir), rpm_dir);

    // CORREÇÃO 4: Garante corte elétrico imediato na ponte H se o alvo for zero
    if (vel_ref_esq == 0.0) pwm_esq = 0;
    if (vel_ref_dir == 0.0) pwm_dir = 0;

    // 4. Aplica os sinais calculados na ponte H
    motorEsq.acionaMotor(pwm_esq);
    motorDir.acionaMotor(pwm_dir);

    vTaskDelay((1000/FREQ_MOTORES_HZ) / portTICK_PERIOD_MS);
  }
}

void processarComando(float comando, float &vel_ref_esq, float &vel_ref_dir, float BASE_RPM) {
  if (comando == 1.0) {         // FRENTE
    vel_ref_esq = BASE_RPM;
    vel_ref_dir = BASE_RPM;
    pidEsq.reset();
    pidDir.reset();
  } 
  else if (comando == -1.0) {   // TRÁS
    vel_ref_esq = -BASE_RPM;
    vel_ref_dir = -BASE_RPM;
    pidEsq.reset();
    pidDir.reset();
  } 
  else if (comando == 2.0) {    // ESQUERDA
    vel_ref_esq = -BASE_RPM;
    vel_ref_dir = BASE_RPM;
    pidEsq.reset();
    pidDir.reset();
  } 
  else if (comando == 3.0) {    // DIREITA
    vel_ref_esq = BASE_RPM;
    vel_ref_dir = -BASE_RPM;
    pidEsq.reset();
    pidDir.reset();
  } 
  else if (comando == 0.0) {    // PARAR
    vel_ref_esq = 0.0;
    vel_ref_dir = 0.0;
    pidEsq.reset();
    pidDir.reset();
  }
}

// Network, display and actuator work never run in the motor/PID task. This
// keeps MQTT connection latency and SPI drawing away from the 50 Hz controller.
void codigoTaskInterface(void * parameter) {
  lockService.begin();

  for (;;) {
    lockService.tick();
    vTaskDelay(pdMS_TO_TICKS(10));
  }
}

// ==========================================
// SETUP PRINCIPAL (O MAESTRO)
// ==========================================
void setup() {
  Serial.begin(921600);
  delay(2000); 
  Serial.println("=== Iniciando Sistema (FreeRTOS) ===");
  filaVelocidadeEsq = xQueueCreate(5, sizeof(float));
  filaVelocidadeDir = xQueueCreate(5, sizeof(float));
  filaComandosSerial = xQueueCreate(5, sizeof(float)); // NOVA FILA

  // 2. Inicializa o Webserver
  webserver_init(WIFI_SSID, WIFI_PASS);

  // 3. Cria Tarefa de Serial (CORE 0 - junto com interface)
  xTaskCreatePinnedToCore(
    codigoTaskSerial,
    "TaskSerial",
    4096,                 // Stack menor para tarefa simples
    NULL,
    2,                    // Prioridade alta para não perder dados
    &TaskSerialCom,
    0                     // CORE 0
  );

  // 4. Cria Tarefa dos Motores (CORE 1)
  xTaskCreatePinnedToCore(
    codigoTaskMotores, 
    "TaskMotores", 
    10000, 
    NULL, 
    1,
    &TaskMotores, 
    1
  );

  // 5. Cria Tarefa de Interface (CORE 0)
  xTaskCreatePinnedToCore(
    codigoTaskInterface,
    "TaskInterface",
    12288,
    NULL,
    1,
    &TaskInterface,
    0
  );

  Serial.println("Tarefas criadas. O FreeRTOS assumiu o controle.");
}

void loop() {
  // Deleta a tarefa padrão do loop para liberar recursos de memória RAM
  vTaskDelete(NULL);
}
