layout(location=0) in vec4 in_v;  // vertex coordinates

uniform mat4 pm;   // projection matrix
uniform mat4 vm;   // view matrix
uniform mat4 mm;   // model matrix

void main() {
    gl_Position = pm * vm * mm * in_v;
}
