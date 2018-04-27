layout(location=0) in vec4 in_v;  // vertex coordinates

uniform mat4 pm;     // projection matrix
uniform mat4 vm;     // view matrix
uniform mat4 mm;     // model matrix
uniform vec3  kd;    // diffuse colour
uniform float alpha; // transparency
out     vec4  v_c;   // vertex colour

void main() {
   v_c = vec4(kd, alpha);
   gl_Position = pm * vm * mm * in_v;
}
