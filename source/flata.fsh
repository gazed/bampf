#version 330

in      vec4  v_c; // color from vertex shader
uniform float fd;  // fade distance
out     vec4  ffc; // final fragment colour

float fade(float distance) {
   float z = gl_FragCoord.z / gl_FragCoord.w / distance;
   z = clamp(z, 0.0, 1.0);
   return 1.0 - z;
}
void main() {
   ffc = v_c;
   ffc.a = ffc.a*fade(fd);
}
