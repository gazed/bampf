in      vec4  v_c;     // color from vertex shader
uniform float fd;      // fade distance
out     vec4  f_color; // final fragment colour

float fade(float distance) {
   float z = gl_FragCoord.z / gl_FragCoord.w / distance;
   z = clamp(z, 0.0, 1.0);
   return 1.0 - z;
}
void main() {
   f_color = v_c;
   f_color.a = f_color.a*fade(fd);
}
