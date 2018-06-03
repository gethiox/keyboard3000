// gyroscope and accelerometer data fetcher for ultimate keyboard3000
// created for STM32 and MPU_6050

#include <Wire.h>

#define MPU_6050     0x68     // MPU default i2c address

// some used registers
#define PWR_MGMT_1   0x6B
#define GYRO_CONFIG  0x1b
#define ACCEL_XOUT_H 0x3b
#define GYRO_XOUT_H  0x43


#define AVERAGE 8             // smoothing gyro output by last X collected values

int PROBES = 1024;            // calibrating gyro by X probes

struct Offset {
  int x;
  int y;
  int z;
} offset;

Offset global_offset = { .x = 0, .y = 0, .z = 0};

void setup() {
//  Serial.println("Ultimate Keyboard3000 gyro-accel addon, v0.0.1 alpha");
    Serial.begin(115200);
//  Serial.println("Enabling Wire");
    Wire.begin();

//  Serial.println("Enabling Gyro...");
    Wire.beginTransmission(MPU_6050);
    Wire.write(PWR_MGMT_1);
    Wire.write(0);          // turn on MPU-6050
    Wire.endTransmission();

//  Serial.println("Set gyro sensivity...");
    Wire.beginTransmission(MPU_6050);
    Wire.write(GYRO_CONFIG);
    Wire.write(2 << 3);     // set scale range to +- 1000 degrees per second
    Wire.endTransmission();
  
//  Serial.println("Give time to boot device up...");
    delay(1000);
//  Serial.println("Calibrating...");
    global_offset = gyro_calc_offset();
}

int16_t acx, acy, acz, tmp, gyx, gyy, gyz;

// calculate gyro offset constant value (aka calibraing)
struct Offset gyro_calc_offset() {
    struct Offset off;

    int gyx_tmp = 0;
    int gyy_tmp = 0;
    int gyz_tmp = 0;

    for (int i = 0; i<=PROBES; i++) {
        Wire.beginTransmission(MPU_6050);
        Wire.write(GYRO_XOUT_H);
        Wire.endTransmission();

        Wire.requestFrom(MPU_6050,6);    // request a total of 6 registers
        gyx=Wire.read()<<8|Wire.read();  // GYRO_XOUT_H, GYRO_XOUT_L
        gyy=Wire.read()<<8|Wire.read();  // GYRO_YOUT_H, GYRO_YOUT_L
        gyz=Wire.read()<<8|Wire.read();  // GYRO_ZOUT_H, GYRO_ZOUT_L

        gyx_tmp += gyx;
        gyy_tmp += gyy;
        gyz_tmp += gyz;
    }

    off.x = gyx_tmp / PROBES;
    off.y = gyy_tmp / PROBES;
    off.z = gyz_tmp / PROBES;

    return off;
}

int16_t gyro_x_array[AVERAGE];
int16_t gyro_y_array[AVERAGE];
int16_t gyro_z_array[AVERAGE];

int16_t accel_x_array[AVERAGE];   
int16_t accel_y_array[AVERAGE];
int16_t accel_z_array[AVERAGE];

int av_counter = 0;

void loop() {
    Wire.beginTransmission(MPU_6050);
    Wire.write(ACCEL_XOUT_H);
    Wire.endTransmission();
    Wire.requestFrom(MPU_6050,14);         // request a total of 14 registers as below

    // raw device values
    acx = Wire.read() << 8 | Wire.read();  // ACCEL_XOUT_H, ACCEL_XOUT_L
    acy = Wire.read() << 8 | Wire.read();  // ACCEL_YOUT_H, ACCEL_YOUT_L
    acz = Wire.read() << 8 | Wire.read();  // ACCEL_ZOUT_H, ACCEL_ZOUT_L
    tmp = Wire.read() << 8 | Wire.read();  // TEMP_OUT_H,   TEMP_OUT_L
    gyx = Wire.read() << 8 | Wire.read();  // GYRO_XOUT_H,  GYRO_XOUT_L)
    gyy = Wire.read() << 8 | Wire.read();  // GYRO_YOUT_H,  GYRO_YOUT_L)
    gyz = Wire.read() << 8 | Wire.read();  // GYRO_ZOUT_H,  GYRO_ZOUT_L)

    // fixing gyro output values by global offset
    gyx -= int16_t(global_offset.x);
    gyy -= int16_t(global_offset.y);
    gyz -= int16_t(global_offset.z);

    // update value arrays for averaging gyro and accel output
    gyro_x_array[av_counter] = gyx;
    gyro_y_array[av_counter] = gyy;
    gyro_z_array[av_counter] = gyz;
    accel_x_array[av_counter] = acx;
    accel_y_array[av_counter] = acy;
    accel_z_array[av_counter] = acz;

    double gyro_x_tmp  = 0;
    double gyro_y_tmp  = 0;
    double gyro_z_tmp  = 0;
    double accel_x_tmp = 0;
    double accel_y_tmp = 0;
    double accel_z_tmp = 0;

    // calculate average output
    for (int i = 0; i< AVERAGE; i++) {
        gyro_x_tmp  += int(gyro_x_array[i]);
        gyro_y_tmp  += int(gyro_y_array[i]);
        gyro_z_tmp  += int(gyro_z_array[i]);
        accel_x_tmp += int(accel_x_array[i]);
        accel_y_tmp += int(accel_y_array[i]);
        accel_z_tmp += int(accel_z_array[i]);
    }

    double avg_gyx = gyro_x_tmp / double(AVERAGE);
    double avg_gyy = gyro_y_tmp / double(AVERAGE);
    double avg_gyz = gyro_z_tmp / double(AVERAGE);
    double avg_acx = accel_x_tmp / double(AVERAGE);
    double avg_acy = accel_y_tmp / double(AVERAGE);
    double avg_acz = accel_z_tmp / double(AVERAGE);

    Serial.print(avg_acx); Serial.print(",");
    Serial.print(avg_acy); Serial.print(",");
    Serial.print(avg_acz); Serial.print(",");
    Serial.print(avg_gyx); Serial.print(",");
    Serial.print(avg_gyy); Serial.print(",");
    Serial.print(avg_gyz); Serial.print(",");
//  Serial.print(tmp / 340.00 + 36.53); // return value in celsius degrees
    Serial.print("\n");
  
    // update counter (index access value of average arrays)
    av_counter += 1;
    av_counter = av_counter % AVERAGE;

//  delay(100); // there is no time for delay, gotta go fast as possible
}

